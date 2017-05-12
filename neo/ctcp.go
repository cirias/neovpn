package neo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

type conn struct {
	gcm cipher.AEAD
	*net.TCPConn
}

func (c *conn) Read(b []byte) (int, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	_, err := io.ReadFull(c.TCPConn, nonce)
	if err != nil {
		return 0, err
	}

	var length uint16
	err = binary.Read(c.TCPConn, binary.BigEndian, &length)
	if err != nil {
		return 0, err
	}

	if len(b) < int(length) {
		return 0, errors.New("ctcp: insufficient buffer size, need " + fmt.Sprintf("%v", length))
	}

	cipherbuf := make([]byte, length)
	_, err = io.ReadFull(c.TCPConn, cipherbuf)
	if err != nil {
		return 0, nil
	}

	buf, err := c.gcm.Open(b[:0], nonce, cipherbuf, nil)

	return len(buf), err
}

func (c *conn) Write(b []byte) (int, error) {
	buf := new(bytes.Buffer)
	nonce := make([]byte, c.gcm.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return 0, err
	}

	_, err = buf.Write(nonce)
	if err != nil {
		return 0, err
	}

	cipherbuf := c.gcm.Seal(nil, nonce, b, nil)

	length := uint16(len(cipherbuf))
	err = binary.Write(buf, binary.BigEndian, length)
	if err != nil {
		return 0, err
	}

	buf.Write(cipherbuf)

	_, err = c.TCPConn.Write(buf.Bytes())

	return len(b), err
}

func newGCM(key string) (cipher.AEAD, error) {
	hash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

func Dial(key, address string) (net.Conn, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	tcpconn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &conn{
		gcm,
		tcpconn.(*net.TCPConn),
	}, nil
}

type listener struct {
	gcm cipher.AEAD
	*net.TCPListener
}

func (ln *listener) Accept() (net.Conn, error) {
	tcpconn, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}

	return &conn{
		ln.gcm,
		tcpconn,
	}, nil
}

func Listen(key, laddr string) (net.Listener, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	tcpln, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	return &listener{
		gcm,
		tcpln.(*net.TCPListener),
	}, nil
}
