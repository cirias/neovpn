package router

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/cirias/neovpn/tunnel"
)

type FakeTun struct {
	c chan struct{}
	io.Writer
}

func (t *FakeTun) Read([]byte) (int, error) {
	<-t.c
	return 0, nil
}

func TestRouter(t *testing.T) {
	psk := "psk"
	laddr := ":9606"
	cidr := "10.10.10.1/30"
	raddr := "127.0.0.1:9606"

	l, err := tunnel.Listen(psk, laddr)
	if err != nil {
		t.Fatal(err)
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatal(err)
	}

	tun := &FakeTun{
		c:      make(chan struct{}),
		Writer: bytes.NewBuffer([]byte{}),
	}

	r := NewRouter(ip, ipNet, tun)

	cc1, err := tunnel.Dial(psk, raddr)
	if err != nil {
		t.Fatal(err)
	}

	sc1, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	go r.Take(sc1)

	if err := cc1.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_REQUEST,
		},
		Payload: []byte("client1"),
	}); err != nil {
		t.Fatal(err)
	}

	pack, err := cc1.Receive()
	if err != nil {
		t.Fatal(err)
	}

	if pack.Header.Type != tunnel.IP_RESPONSE {
		t.Error("wrong pack type:", pack.Header.Type, ", expect IP_RESPONSE")
	}

	if pack.Payload[0] != 10 ||
		pack.Payload[1] != 10 ||
		pack.Payload[2] != 10 ||
		pack.Payload[3] != 2 {
		t.Error("wrong payload: ip address:", pack.Payload[0:4])
	}

	// etc

	cc2, err := tunnel.Dial(psk, raddr)
	if err != nil {
		t.Fatal(err)
	}

	sc2, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	go r.Take(sc2)

	if err := cc2.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_REQUEST,
		},
		Payload: []byte("client2"),
	}); err != nil {
		t.Fatal(err)
	}

	pack, err = cc2.Receive()
	if err != nil {
		t.Fatal(err)
	}

	if pack.Header.Type != tunnel.IP_RESPONSE {
		t.Error("wrong pack type:", pack.Header.Type, ", expect IP_RESPONSE")
	}

	if pack.Payload[0] != 10 ||
		pack.Payload[1] != 10 ||
		pack.Payload[2] != 10 ||
		pack.Payload[3] != 3 {
		t.Error("wrong payload: ip address:", pack.Payload[0:4])
	}
}
