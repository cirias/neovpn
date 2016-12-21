package main

import (
	"log"

	"github.com/cirias/neovpn/server/router"
	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

func main() {
	l, err := tunnel.Listen("psk unused", ":9606")
	if err != nil {
		log.Fatal(err)
	}

	t, err := tun.NewTUN("")
	if err != nil {
		log.Fatal(err)
	}

	r := router.NewRouter([]byte{10, 10, 10, 1}, 2, t)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go r.Take(c)
	}
}
