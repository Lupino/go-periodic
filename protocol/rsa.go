package protocol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
)

// Corresponds to Haskell's RSAMode
const (
	ModePlain = 0
	ModeRSA   = 1
	ModeAES   = 2
)

const (
	aesKeySize = 32
	aesIvSize  = 16
	oaepSize   = 66 // SHA256 overhead
)

type RSAConn struct {
	net.Conn
	mode       int
	privateKey *rsa.PrivateKey
	peerPub    *rsa.PublicKey
	sessionKey []byte

	// Internal buffer for processing streaming data after AES/RSA decryption
	readBuf bytes.Buffer
}

type RSAConnParam struct {
	PrivateKeyPath      string
	ServerPublicKeyPath string
	Mode                int
}

// -----------------------------------------------------------------------------
// Constructor and Handshake
// -----------------------------------------------------------------------------

// NewClientRSAConn creates a client connection
func NewClientRSAConn(conn net.Conn, param RSAConnParam) (*RSAConn, error) {
	// [Check 1] Validate local parameter mode before starting
	switch param.Mode {
	case ModePlain, ModeRSA, ModeAES:
		// Valid
	default:
		return nil, fmt.Errorf("invalid client mode parameter: %d", param.Mode)
	}

	privKey, err := loadPrivateKey(param.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	serverPub, err := loadPublicKey(param.ServerPublicKeyPath)
	if err != nil {
		return nil, err
	}

	rc := &RSAConn{
		Conn:       conn,
		privateKey: privKey,
		peerPub:    serverPub,
		readBuf:    bytes.Buffer{},
	}

	// 1. Identity handshake
	if err := rc.clientHandshake(); err != nil {
		return nil, fmt.Errorf("handshake failed: %v", err)
	}

	// 2. Mode negotiation
	modeByte := []byte{byte(param.Mode)}
	if err := rc.sendDataOAEP(modeByte); err != nil {
		return nil, fmt.Errorf("failed to send mode: %v", err)
	}
	rc.mode = param.Mode

	// 3. Key exchange (Required only for AES mode)
	if param.Mode == ModeAES {
		key := make([]byte, aesKeySize)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("failed to gen session key: %v", err)
		}
		rc.sessionKey = key
		if err := rc.sendDataOAEP(key); err != nil {
			return nil, fmt.Errorf("failed to send session key: %v", err)
		}
	}

	return rc, nil
}

// NewServerRSAConn creates a server connection
func NewServerRSAConn(conn net.Conn, privKeyPath, authorizedKeysDir string) (*RSAConn, error) {
	privKey, err := loadPrivateKey(privKeyPath)
	if err != nil {
		return nil, err
	}

	authKeys, err := loadPublicKeysFromDir(authorizedKeysDir)
	if err != nil {
		return nil, err
	}

	rc := &RSAConn{
		Conn:       conn,
		privateKey: privKey,
		readBuf:    bytes.Buffer{},
	}

	// 1. Identity handshake
	clientPub, err := rc.serverHandshake(authKeys)
	if err != nil {
		return nil, fmt.Errorf("handshake failed: %v", err)
	}
	rc.peerPub = clientPub

	// 2. Receive mode
	modeBytes, err := rc.recvDataOAEP()
	if err != nil {
		return nil, fmt.Errorf("failed to recv mode: %v", err)
	}

	if len(modeBytes) == 0 {
		return nil, errors.New("received empty mode data")
	}

	// [Check 2] Validate received mode from client
	reqMode := int(modeBytes[0])
	switch reqMode {
	case ModePlain, ModeRSA, ModeAES:
		rc.mode = reqMode
	default:
		return nil, fmt.Errorf("received invalid mode from client: %d", reqMode)
	}

	// 3. Receive key (if client requests AES)
	if rc.mode == ModeAES {
		key, err := rc.recvDataOAEP()
		if err != nil {
			return nil, fmt.Errorf("failed to recv session key: %v", err)
		}
		if len(key) != aesKeySize {
			return nil, fmt.Errorf("invalid session key size: got %d, expected %d", len(key), aesKeySize)
		}
		rc.sessionKey = key
	}

	return rc, nil
}

// -----------------------------------------------------------------------------
// Read / Write Implementation
// -----------------------------------------------------------------------------

func (c *RSAConn) Read(b []byte) (n int, err error) {
	if c.readBuf.Len() > 0 {
		return c.readBuf.Read(b)
	}

	switch c.mode {
	case ModePlain:
		return c.Conn.Read(b)

	case ModeRSA:
		data, err := c.recvDataOAEP()
		if err != nil {
			return 0, err
		}
		c.readBuf.Write(data)

	case ModeAES:
		data, err := c.recvDataAES()
		if err != nil {
			return 0, err
		}
		c.readBuf.Write(data)
	default:
		// Should not happen if handshake validated correctly
		return 0, fmt.Errorf("conn in invalid mode: %d", c.mode)
	}

	return c.readBuf.Read(b)
}

func (c *RSAConn) Write(b []byte) (n int, err error) {
	switch c.mode {
	case ModePlain:
		return c.Conn.Write(b)

	case ModeRSA:
		if err := c.sendDataOAEP(b); err != nil {
			return 0, err
		}
		return len(b), nil

	case ModeAES:
		if err := c.sendDataAES(b); err != nil {
			return 0, err
		}
		return len(b), nil
	default:
		return 0, fmt.Errorf("conn in invalid mode: %d", c.mode)
	}
}

// -----------------------------------------------------------------------------
// Core Transport Logic
// -----------------------------------------------------------------------------

func (c *RSAConn) recvDataAES() ([]byte, error) {
	lenBuf := make([]byte, 8)
	if _, err := io.ReadFull(c.Conn, lenBuf); err != nil {
		return nil, err
	}
	pktLen := int(binary.BigEndian.Uint64(lenBuf))

	if pktLen < aesIvSize || pktLen > 100*1024*1024 {
		return nil, fmt.Errorf("invalid packet length: %d", pktLen)
	}

	payload := make([]byte, pktLen)
	if _, err := io.ReadFull(c.Conn, payload); err != nil {
		return nil, err
	}

	iv := payload[:aesIvSize]
	ciphertext := payload[aesIvSize:]

	block, err := aes.NewCipher(c.sessionKey)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

func (c *RSAConn) sendDataAES(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	iv := make([]byte, aesIvSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	block, err := aes.NewCipher(c.sessionKey)
	if err != nil {
		return err
	}
	stream := cipher.NewCTR(block, iv)
	ciphertext := make([]byte, len(data))
	stream.XORKeyStream(ciphertext, data)

	payload := append(iv, ciphertext...)
	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, uint64(len(payload)))

	finalPkt := append(lenBuf, payload...)
	if _, err := c.Conn.Write(finalPkt); err != nil {
		return err
	}
	return nil
}

func (c *RSAConn) recvDataOAEP() ([]byte, error) {
	size := c.privateKey.Size()
	buf := make([]byte, size)

	if _, err := io.ReadFull(c.Conn, buf); err != nil {
		return nil, err
	}

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, c.privateKey, buf, nil)
	if err != nil {
		return nil, fmt.Errorf("OAEP decrypt error: %v", err)
	}
	return plaintext, nil
}

func (c *RSAConn) sendDataOAEP(data []byte) error {
	msgLen := len(data)
	step := c.peerPub.Size() - oaepSize
	if step <= 0 {
		return errors.New("key size too small for OAEP SHA256")
	}

	for i := 0; i < msgLen; i += step {
		end := i + step
		if end > msgLen {
			end = msgLen
		}
		chunk := data[i:end]

		cipherText, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, c.peerPub, chunk, nil)
		if err != nil {
			return err
		}
		if _, err := c.Conn.Write(cipherText); err != nil {
			return err
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Handshake Helper Functions
// -----------------------------------------------------------------------------

func (c *RSAConn) clientHandshake() error {
	fp := publicKeyFingerprint(&c.privateKey.PublicKey)
	if err := c.sendDataOAEP(fp); err != nil {
		return err
	}

	serverFp, err := c.recvDataOAEP()
	if err != nil {
		return err
	}

	expectedFp := publicKeyFingerprint(c.peerPub)
	if !bytes.Equal(serverFp, expectedFp) {
		return fmt.Errorf("server fingerprint mismatch")
	}
	return nil
}

func (c *RSAConn) serverHandshake(allowedKeys []*rsa.PublicKey) (*rsa.PublicKey, error) {
	clientFp, err := c.recvDataOAEP()
	if err != nil {
		return nil, err
	}

	var matchedKey *rsa.PublicKey
	for _, key := range allowedKeys {
		if bytes.Equal(publicKeyFingerprint(key), clientFp) {
			matchedKey = key
			break
		}
	}

	if matchedKey == nil {
		return nil, errors.New("client unknown (fingerprint not in authorized_keys)")
	}

	c.peerPub = matchedKey
	myFp := publicKeyFingerprint(&c.privateKey.PublicKey)
	if err := c.sendDataOAEP(myFp); err != nil {
		return nil, err
	}

	return matchedKey, nil
}

// -----------------------------------------------------------------------------
// Crypto Utilities
// -----------------------------------------------------------------------------

func publicKeyFingerprint(pub *rsa.PublicKey) []byte {
	der := x509.MarshalPKCS1PublicKey(pub)
	h := sha256.Sum256(der)
	return h[:]
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("no PEM data found in private key file")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parsePublicKey(raw)
}

func loadPublicKeysFromDir(dir string) ([]*rsa.PublicKey, error) {
	var keys []*rsa.PublicKey
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if pk, err2 := loadPublicKey(dir); err2 == nil {
			return []*rsa.PublicKey{pk}, nil
		}
		return nil, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".pem") {
			path := filepath.Join(dir, f.Name())
			raw, err := ioutil.ReadFile(path)
			if err == nil {
				if pk, err := parsePublicKey(raw); err == nil {
					keys = append(keys, pk)
				}
			}
		}
	}
	return keys, nil
}

func parsePublicKey(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	return nil, errors.New("unknown public key format")
}
