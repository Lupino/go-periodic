package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

// ClientType Define the client type.
type ClientType int

const (
	// TYPECLIENT defined the connection client is a client.
	TYPECLIENT ClientType = iota + 1
	// TYPEWORKER defined the connection client is a worker.
	TYPEWORKER
	// TYPEAUTHCLIENT defined the authenticated connection client is a client.
	TYPEAUTHCLIENT
	// TYPEAUTHWORKER defined the authenticated connection client is a worker.
	TYPEAUTHWORKER
)

// ClientAuth defines the authenticated client identity sent during registration.
type ClientAuth struct {
	Name  string
	Token string
}

// Bytes convert client type to Byte
func (c ClientType) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(c))
	return buf.Bytes()
}

func encodeBinaryByteString(data []byte) []byte {
	buf := bytes.NewBuffer(nil)
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(data)))
	buf.Write(size[:])
	buf.Write(data)
	return buf.Bytes()
}

// Bytes converts the client auth identity to the Haskell Binary ByteString format.
func (a ClientAuth) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(encodeBinaryByteString([]byte(a.Name)))
	buf.Write(encodeBinaryByteString([]byte(a.Token)))
	return buf.Bytes()
}

// RegistrationBytes converts the client type and optional auth identity to the
// periodic registration packet payload.
func (c ClientType) RegistrationBytes(auth *ClientAuth) ([]byte, error) {
	if auth == nil {
		return c.Bytes(), nil
	}
	if auth.Name == "" || auth.Token == "" {
		return nil, fmt.Errorf("auth name and token must be provided together")
	}
	buf := bytes.NewBuffer(nil)
	switch c {
	case TYPECLIENT:
		buf.WriteByte(byte(TYPEAUTHCLIENT))
	case TYPEWORKER:
		buf.WriteByte(byte(TYPEAUTHWORKER))
	default:
		return nil, fmt.Errorf("unsupported auth client type %d", c)
	}
	buf.Write(auth.Bytes())
	return buf.Bytes(), nil
}

// to string `TYPECLIENT`, `TYPEWORKER`.
func (c ClientType) String() string {
	switch c {
	case TYPECLIENT:
		return "TYPECLIENT"
	case TYPEWORKER:
		return "TYPEWORKER"
	case TYPEAUTHCLIENT:
		return "TYPEAUTHCLIENT"
	case TYPEAUTHWORKER:
		return "TYPEAUTHWORKER"
	}
	panic("Unknow ClientType " + strconv.Itoa(int(c)))
}
