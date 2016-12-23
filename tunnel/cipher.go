package tunnel

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

const (
	IVSize = aes.BlockSize
)

type crypter struct {
	block cipher.Block
}

func newCrypter(key []byte) (*crypter, error) {
	hash := sha256.Sum256(key)

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	return &crypter{block}, nil
}

func newIV() ([]byte, error) {
	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	return iv, nil
}

type decrypter struct {
	stream cipher.Stream
}

func newDecrypter(c *crypter, iv []byte) *decrypter {
	return &decrypter{
		stream: cipher.NewOFB(c.block, iv),
	}
}

func (dec *decrypter) Decrypt(dst, src []byte) {
	dec.stream.XORKeyStream(dst, src)
}

type encrypter struct {
	stream cipher.Stream
}

func newEncrypter(c *crypter, iv []byte) *encrypter {
	return &encrypter{
		stream: cipher.NewOFB(c.block, iv),
	}
}

func (enc *encrypter) Encrypt(dst, src []byte) {
	enc.stream.XORKeyStream(dst, src)
}
