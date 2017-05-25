package main

import (
	"flag"
	"log"
	"net"
	"sync"
)

func main() {
	var key, sAddr, lAddr, ipAddr string
	flag.StringVar(&key, "key", "", "pre-shared key")
	flag.StringVar(&sAddr, "server", "", "server address for client mode")
	flag.StringVar(&lAddr, "listen", "", "listen address for server mode")
	flag.StringVar(&ipAddr, "ip", "", "ip address in CIDR")
	flag.Parse()

	if sAddr != "" {
		client(key, sAddr, ipAddr)
	} else if lAddr != "" {
		server(key, lAddr, ipAddr)
	}
}

func client(key, rAddr, ipAddr string) {
	conn, err := Dial(key, rAddr)
	if err != nil {
		log.Fatalf("could not dial to %v@%v: %v\n", key, rAddr, err)
	}
	defer func() { _ = conn.Close() }()

	tun, err := newTun()
	if err != nil {
		log.Fatalf("could not new tun: %v\n", err)
	}
	defer func() { _ = tun.Close() }()

	if err := ifUp(tun.Name(), ipAddr); err != nil {
		log.Fatalf("could not turn up interface: %v", err)
	}
	defer func() {
		if err := ifDown(tun.Name(), ipAddr); err != nil {
			log.Fatalf("could not turn down interface: %v", err)
		}
	}()

	c := newConnector(tun, conn)
	defer c.Close()

	for err := range c.Run() {
		log.Fatalf("could not run: %v\n", err)
	}
}

func server(key, lAddr, ipAddr string) {
	tun, err := newTun()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() { _ = tun.Close() }()

	if err := ifUp(tun.Name(), ipAddr); err != nil {
		log.Fatalf("could not turn up interface: %v", err)
	}
	defer func() {
		if err := ifDown(tun.Name(), ipAddr); err != nil {
			log.Fatalf("could not turn down interface: %v", err)
		}
	}()

	ln, err := Listen(key, lAddr)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() { _ = ln.Close() }()

	conns := make(map[[4]byte]net.Conn)
	var connsmtx sync.RWMutex

	go func() {
		var dst [4]byte

		for {
			b := make([]byte, 65535)
			n, err := tun.Read(b)
			if err != nil {
				log.Fatalln(err)
			}

			copy(dst[:], b[16:20])

			if net.IPv4(dst[0], dst[1], dst[2], dst[3]).Equal(net.IPv4bcast) {
				for _, conn := range conns {
					if _, err := conn.Write(b[:n]); err != nil {
						log.Println(err)
					}
				}
			}

			connsmtx.RLock()
			conn, ok := conns[dst]
			connsmtx.RUnlock()
			if !ok {
				log.Println("not found: dst:", dst)
				continue
			}

			_, err = conn.Write(b[:n])
			if err != nil {
				log.Println(err)
			}
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			defer func() { _ = conn.Close() }()

			var src [4]byte
			for {
				b := make([]byte, 65535)
				n, err := conn.Read(b)
				if err != nil {
					log.Println(err)
					break
				}

				copy(src[:], b[12:16])

				connsmtx.RLock()
				storedConn, ok := conns[src]
				connsmtx.RUnlock()

				if !ok || storedConn != conn {
					connsmtx.Lock()
					conns[src] = conn
					connsmtx.Unlock()
				}

				_, err = tun.Write(b[:n])
				if err != nil {
					log.Println(err)
					break
				}
			}
		}()
	}
}
