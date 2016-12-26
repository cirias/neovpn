package router

import (
	"net"
	"testing"

	"github.com/cirias/neovpn/server/router"
	"github.com/cirias/neovpn/tunnel"
)

func TestRouter(t *testing.T) {
	l, err := tunnel.Listen(psk, laddr)
	if err != nil {
		t.Fatal(err)
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatal(err)
	}

	r := router.NewRouter(ip, ipNet, t)

	cc, err := tunnel.Dial(psk, raddr)
	if err != nil {
		t.Fatal(err)
	}

	sc, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}

	go r.Take(sc)

	if err := cc.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_REQUEST,
		},
		Payload: []byte("client1"),
	}); err != nil {
		return err
	}

	pack, err := cc.Receive()
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

	if err := cc.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_REQUEST,
		},
		Payload: []byte("client2"),
	}); err != nil {
		return err
	}

	pack, err := cc.Receive()
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
