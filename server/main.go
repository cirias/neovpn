package main

import (
	"log"
	"net"

	"github.com/cirias/neovpn/server/router"
	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

func main() {
	l, err := tunnel.Listen("psk unused", ":9606")
	if err != nil {
		log.Fatal(err)
	}

	t, err := tun.NewTUN("", "./up.sh", "./down.sh")
	if err != nil {
		log.Fatal(err)
	}

	cidr := "10.10.10.1/30"
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatal(err)
	}

	r := router.NewRouter(ip, ipNet, t)
	t.Up(ip, ipNet)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go r.Take(c)
	}
}
