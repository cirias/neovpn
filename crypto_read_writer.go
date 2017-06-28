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
)

type cryptoReadWriter struct {
	gcm cipher.AEAD
	rwc io.ReadWriteCloser
}

func newCryptoReadWriter(rwc io.ReadWriteCloser, key string) (io.ReadWriter, error) {
	hash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, fmt.Errorf("could not new cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("could not new GCM: %v", err)
	}

	return &cryptoReadWriter{
		gcm,
		rwc,
	}, nil
}

func (c *cryptoReadWriter) Read(b []byte) (int, error) {
	nonce := make([]byte, c.gcm.NonceSize())
	_, err := io.ReadFull(c.rwc, nonce)
	if err != nil {
		return 0, fmt.Errorf("could not read nonce: %v", err)
	}

	var length uint16
	err = binary.Read(c.rwc, binary.BigEndian, &length)
	if err != nil {
		return 0, fmt.Errorf("could not read length: %v", err)
	}

	if len(b) < int(length) {
		return 0, errors.New("insufficient buffer size, need " + fmt.Sprintf("%v", length))
	}

	cipherbuf := make([]byte, length)
	_, err = io.ReadFull(c.rwc, cipherbuf)
	if err != nil {
		return 0, fmt.Errorf("could not read cipherbuf: %v", err)
	}

	buf, err := c.gcm.Open(b[:0], nonce, cipherbuf, nil)
	if err != nil {
		return 0, fmt.Errorf("could not open cipherbuf: %v", err)
	}

	return len(buf), nil
}

func (c *cryptoReadWriter) Write(b []byte) (int, error) {
	buf := new(bytes.Buffer)
	nonce := make([]byte, c.gcm.NonceSize())
	_, err := rand.Read(nonce)
	if err != nil {
		return 0, fmt.Errorf("could not create random nonce: %v", err)
	}

	_, err = buf.Write(nonce)
	if err != nil {
		return 0, fmt.Errorf("could not write nonce: %v", err)
	}

	cipherbuf := c.gcm.Seal(nil, nonce, b, nil)

	length := uint16(len(cipherbuf))
	err = binary.Write(buf, binary.BigEndian, length)
	if err != nil {
		return 0, fmt.Errorf("could not write length: %v", err)
	}

	buf.Write(cipherbuf)

	_, err = c.rwc.Write(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("could not write cipherbuf: %v", err)
	}

	return len(b), err
}

func (c *cryptoReadWriter) Close() error {
	return c.rwc.Close()
}
