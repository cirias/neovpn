package tunnel

import (
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

	return &Conn{l.psk, conn}, nil
}

func Dial(psk, raddr string) (*Conn, error) {
	log.Println("Dial", raddr)
	conn, err := net.Dial("tcp", raddr)
	if err != nil {
		return nil, err
	}

	return &Conn{psk, conn}, nil
}

func (c *Conn) Receive() (*Pack, error) {
	h := &Header{}
	if err := binary.Read(c, binary.BigEndian, h); err != nil {
		return nil, err
	}

	p := make([]byte, h.Len)
	if _, err := io.ReadFull(c, p); err != nil {
		return nil, err
	}

	return &Pack{h, p}, nil
}

func (c *Conn) Send(p *Pack) error {
	p.Header.Len = int16(len(p.Payload))
	if err := binary.Write(c, binary.BigEndian, p.Header); err != nil {
		return err
	}

	if _, err := c.Write(p.Payload); err != nil {
		return err
	}

	return nil
}
