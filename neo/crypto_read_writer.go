package main

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
	"log"
)

type conn struct {
	gcm cipher.AEAD
	rw  io.ReadWriter
}

func (c *conn) Read(b []byte) (int, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	_, err := io.ReadFull(c.rw, nonce)
	if err != nil {
		return 0, err
	}

	var length uint16
	err = binary.Read(c.rw, binary.BigEndian, &length)
	if err != nil {
		return 0, err
	}

	if len(b) < int(length) {
		return 0, errors.New("ctcp: insufficient buffer size, need " + fmt.Sprintf("%v", length))
	}

	cipherbuf := make([]byte, length)
	_, err = io.ReadFull(c.rw, cipherbuf)
	if err != nil {
		return 0, nil
	}

	buf, err := c.gcm.Open(b[:0], nonce, cipherbuf, nil)

	log.Println("conn.Read:", b[:len(buf)])
	return len(buf), err
}

func (c *conn) Write(b []byte) (int, error) {
	log.Println("conn.Write:", b)
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

	_, err = c.rw.Write(buf.Bytes())

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

func NewCryptoReadWriter(rw io.ReadWriter, key string) (io.ReadWriter, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	return &conn{
		gcm,
		rw,
	}, nil
}
