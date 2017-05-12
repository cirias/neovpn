package main

import (
	"flag"
	"log"
	"net"
	"sync"
)

func main() {
	var key, saddr, laddr string
	flag.StringVar(&key, "key", "", "pre-shared key")
	flag.StringVar(&saddr, "server", "", "server address for client mode")
	flag.StringVar(&laddr, "listen", "", "listen address for server mode")
	flag.Parse()

	if saddr != "" {
		client(key, saddr)
	} else if laddr != "" {
		server(key, laddr)
	}
}

func client(key, address string) {
	tun, err := newTun()
	if err != nil {
		log.Fatalln(err)
	}
	defer tun.Close()

	conn, err := Dial(key, address)
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	go func() {
		b := make([]byte, 65535)

		for {
			n, err := tun.Read(b)
			if err != nil {
				log.Fatalln(err)
			}

			_, err = conn.Write(b[:n])
			if err != nil {
				log.Fatalln(err)
			}
		}
	}()

	{
		b := make([]byte, 65535)

		for {
			n, err := conn.Read(b)
			if err != nil {
				log.Fatalln(err)
			}

			_, err = tun.Write(b[:n])
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
}

func server(key, address string) {
	tun, err := newTun()
	if err != nil {
		log.Fatalln(err)
	}
	defer tun.Close()

	ln, err := Listen(key, address)
	if err != nil {
		log.Fatalln(err)
	}
	defer ln.Close()

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
			defer conn.Close()

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
