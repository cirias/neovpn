package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
)

func main() {
	var key, sAddr, lAddr, ipAddr string
	flag.StringVar(&key, "key", "", "pre-shared key")
	flag.StringVar(&sAddr, "server", "", "server address for client mode")
	flag.StringVar(&lAddr, "listen", "", "listen address for server mode")
	flag.StringVar(&ipAddr, "ip", "", "ip address in CIDR for local interface")
	flag.Parse()

	if sAddr != "" {
		if err := client(key, sAddr, ipAddr); err != nil {
			log.Fatalln(err)
		}
	} else if lAddr != "" {
		if err := server(key, lAddr, ipAddr); err != nil {
			log.Fatalln(err)
		}
	}
}

func client(key, rAddr, ipAddr string) error {
	conn, err := Dial(key, rAddr)
	if err != nil {
		return fmt.Errorf("could not dial to %v@%v: %v", key, rAddr, err)
	}
	defer func() { _ = conn.Close() }()

	tun, err := newTun()
	if err != nil {
		return fmt.Errorf("could not new tun: %v", err)
	}
	defer func() { _ = tun.Close() }()

	down, err := up(tun.Name(), addIP(ipAddr))
	if err != nil {
		return fmt.Errorf("could not turn interface up: %v", err)
	}
	defer func() {
		if err := down(); err != nil {
			log.Printf("could not turn interface down: %v\n", err)
		}
	}()

	if err := <-pipe(tun, conn); err != nil {
		return fmt.Errorf("could not pipe tun and conn: %v\n", err)
	}

	return nil
}

func server(key, lAddr, ipAddr string) error {
	tun, err := newTun()
	if err != nil {
		return fmt.Errorf("could not new tun: %v", err)
	}
	defer func() { _ = tun.Close() }()

	down, err := up(tun.Name(), addIP(ipAddr))
	if err != nil {
		return fmt.Errorf("could not turn interface up: %v", err)
	}
	defer func() {
		if err := down(); err != nil {
			log.Printf("could not turn interface down: %v\n", err)
		}
	}()

	ln, err := Listen(key, lAddr)
	if err != nil {
		return fmt.Errorf("could not listen to %v@%v: %v", key, lAddr, err)
	}
	log.Printf("listening to %v@%v", key, lAddr)
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
