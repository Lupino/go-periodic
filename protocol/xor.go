package protocol

import (
	"net"
)

// XORConn a custom connect
type XORConn struct {
	net.Conn
	key  []byte
	size int
	rx   int
	wx   int
}

// NewXORConn create a connection
func NewXORConn(conn net.Conn, key []byte) *XORConn {
	return &XORConn{Conn: conn, key: key, size: len(key), rx: 0, wx: 0}
}

func (conn *XORConn) Read(b []byte) (n int, err error) {
	if n, err = conn.Conn.Read(b); err != nil {
		return
	}
	for i := 0; i < n; i++ {
		b[i] = b[i] ^ conn.key[conn.rx%conn.size]
		conn.rx++
	}
	return
}

func (conn *XORConn) Write(b []byte) (n int, err error) {
	size := len(b)
	enc := make([]byte, size)
	for i := 0; i < size; i++ {
		enc[i] = b[i] ^ conn.key[conn.wx%conn.size]
		conn.wx++
	}
	written := 0
	for written < size {
		wrote, err := conn.Conn.Write(enc[written:])
		if err != nil {
			return 0, err
		}
		written = written + wrote
	}
	return size, nil
}
