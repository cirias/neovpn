package tunnel

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
)

const (
	IP_REQUEST = iota
	IP_RESPONSE
	IP_PACKET
)

type Listener struct {
	psk string
	net.Listener
}

type Conn struct {
	psk string
	net.Conn
	crypter *crypter
}

type Header struct {
	Len  int16 // length of payload
	Type byte
}

type Pack struct {
	*Header
	Payload []byte
}

func Listen(psk, laddr string) (*Listener, error) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	return &Listener{psk, l}, nil
}

func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return newConn(l.psk, conn)
}

func Dial(psk, raddr string) (*Conn, error) {
	log.Println("Dial", raddr)
	conn, err := net.Dial("tcp", raddr)
	if err != nil {
		return nil, err
	}

	return newConn(psk, conn)
}

func newConn(psk string, c net.Conn) (*Conn, error) {
	crypter, err := newCrypter([]byte(psk))
	if err != nil {
		return nil, err
	}

	return &Conn{psk, c, crypter}, nil
}

func (c *Conn) Receive() (*Pack, error) {
	// IV
	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(c, iv); err != nil {
		return nil, err
	}

	dec := newDecrypter(c.crypter, iv)

	// Header
	h := &Header{}
	eh := make([]byte, binary.Size(h))
	if _, err := io.ReadFull(c, eh); err != nil {
		return nil, err
	}

	rh := make([]byte, len(eh))
	dec.Decrypt(rh, eh)

	if err := binary.Read(bytes.NewBuffer(rh), binary.BigEndian, h); err != nil {
		return nil, err
	}

	// Payload
	ep := make([]byte, h.Len)
	if _, err := io.ReadFull(c, ep); err != nil {
		return nil, err
	}

	p := make([]byte, len(ep))
	dec.Decrypt(p, ep)

	return &Pack{h, p}, nil
}

func (c *Conn) Send(p *Pack) error {
	iv, err := newIV()
	if err != nil {
		return err
	}

	enc := newEncrypter(c.crypter, iv)

	var buf bytes.Buffer

	p.Header.Len = int16(len(p.Payload))
	if err := binary.Write(&buf, binary.BigEndian, p.Header); err != nil {
		return err
	}

	if _, err := buf.Write(p.Payload); err != nil {
		return err
	}

	ep := make([]byte, buf.Len())
	enc.Encrypt(ep, buf.Bytes())

	if _, err := c.Write(bytes.Join([][]byte{iv, ep}, nil)); err != nil {
		return err
	}

	return nil
}
