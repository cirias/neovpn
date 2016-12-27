package main

import (
	"flag"
	"log"
	"net"

	"github.com/cirias/neovpn/server/router"
	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

func main() {
	var cidr, psk, laddr, upScript, downScript string

	flag.StringVar(&cidr, "cidr", "10.10.10.1/30", "vpn cidr")
	flag.StringVar(&psk, "psk", "", "pre-shared key")
	flag.StringVar(&laddr, "laddr", ":9606", "listening address")
	flag.StringVar(&upScript, "up-script", "./up.sh", "up hook script path")
	flag.StringVar(&downScript, "down-script", "./down.sh", "down hook script path")
	flag.Parse()

	l, err := tunnel.Listen(psk, laddr)
	if err != nil {
		log.Fatal(err)
	}

	t, err := tun.NewTUN("", upScript, downScript)
	if err != nil {
		log.Fatal(err)
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatal(err)
	}

	output, err := t.Up(ip, ipNet)
	if err != nil {
		log.Fatalln("tun up:", err)
	} else if len(output) != 0 {
		log.Println("tun up:", string(output))
	}

	r := router.NewRouter(ip, ipNet, t)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go r.Take(c)
	}
}
