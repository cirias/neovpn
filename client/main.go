package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

func main() {
	id := "id"
	psk := "psk"
	raddr := "server:9606"

	t, err := tun.NewTUN("")
	if err != nil {
		log.Fatal(err)
	}

	var c *tunnel.Conn

	go func() {
		for {
			c, err = tunnel.Dial(psk, raddr)
			if err != nil {
				log.Println(err)
				continue
			}

			if err := runConn(id, c, t); err != nil {
				log.Println(err)
				continue
			}
		}
	}()

	for {
		ipPacket, err := t.Read()
		if err != nil {
			log.Println(err)
			continue
		}

		if c == nil {
			continue
		}

		if err := c.Send(&tunnel.Pack{
			Header: &tunnel.Header{
				Type: tunnel.IP_PACKET,
			},
			Payload: ipPacket,
		}); err != nil {
			log.Println(err)
		}
	}
}

func runConn(id string, c *tunnel.Conn, t *tun.Tun) error {
	defer c.Close()

	if err := c.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_REQUEST,
		},
		Payload: []byte(id),
	}); err != nil {
		return err
	}

	for {
		pack, err := c.Receive()
		if err != nil {
			return err
		}

		switch pack.Header.Type {
		case tunnel.IP_RESPONSE:
			log.Println("receive IP", pack.Payload)
			t.Setup(pack.Payload)
		case tunnel.IP_PACKET:
			log.Println("receive packet", pack.Payload)
			if _, err := t.Write(pack.Payload); err != nil {
				return err
			}
		default:
			return errors.New("invalid type: " + fmt.Sprint(pack.Header.Type))
		}
	}
}
