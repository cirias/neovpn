package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

func main() {
	var id, psk, raddr, upScript, downScript string

	flag.StringVar(&id, "id", "", "client ID")
	flag.StringVar(&psk, "psk", "", "pre-shared key")
	flag.StringVar(&raddr, "raddr", ":9606", "remote server address")
	flag.StringVar(&upScript, "up-script", "./up.sh", "up hook script path")
	flag.StringVar(&downScript, "down-script", "./down.sh", "down hook script path")
	flag.Parse()

	t, err := tun.NewTUN("", upScript, downScript)
	if err != nil {
		log.Fatal(err)
	}

	var c *tunnel.Conn

	go func() {
		for {
			log.Println("dial:", raddr)
			c, err = tunnel.Dial(psk, raddr)
			if err != nil {
				log.Println(err)
				time.Sleep(2 * time.Second)
				continue
			}

			if err := runConn(id, c, t); err != nil {
				log.Println(err)
				time.Sleep(2 * time.Second)
				continue
			}
		}
	}()

	for {
		ipPacket := make([]byte, tun.MAX_PACKET_SIZE)
		n, err := t.Read(ipPacket)
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
			Payload: ipPacket[:n],
		}); err != nil {
			log.Println(err)
		}
	}
}

func runConn(id string, c *tunnel.Conn, i tun.Interface) error {
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
			ip := net.IP(pack.Payload[0:4])
			ipNet := &net.IPNet{
				IP:   pack.Payload[4:8],
				Mask: pack.Payload[8:12],
			}
			log.Println("receive IP", ip, ipNet)
			output, err := i.Up(ip, ipNet)
			if err != nil {
				log.Println("tun up:", err)
			} else if len(output) != 0 {
				log.Println("tun up:", string(output))
			}

		case tunnel.IP_PACKET:
			// log.Println("receive packet", pack.Payload)
			if _, err := i.Write(pack.Payload); err != nil {
				return err
			}
		default:
			return errors.New("invalid type: " + fmt.Sprint(pack.Header.Type))
		}
	}
}
