package main

import (
	"flag"
	"fmt"
	"io"
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
	conn, err := net.Dial("tcp", rAddr)
	if err != nil {
		return fmt.Errorf("could not dial to %v@%v: %v", key, rAddr, err)
	}
	defer func() { _ = conn.Close() }()

	crw, err := NewCryptoReadWriter(conn, key)
	if err != nil {
		return fmt.Errorf("could not new CryptoReadWriter: %v", err)
	}

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

	if err := <-pipe(tun, crw); err != nil {
		return fmt.Errorf("could not pipe tun and crw: %v\n", err)
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

	ln, err := net.Listen("tcp", lAddr)
	if err != nil {
		return fmt.Errorf("could not listen to %v@%v: %v", key, lAddr, err)
	}
	log.Printf("listening to %v@%v", key, lAddr)
	defer func() { _ = ln.Close() }()

	crws := make(map[[4]byte]io.ReadWriter)
	var mtx sync.RWMutex

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
				for _, crw := range crws {
					if _, err := crw.Write(b[:n]); err != nil {
						log.Println(err)
					}
				}
			}

			mtx.RLock()
			crw, ok := crws[dst]
			mtx.RUnlock()
			if !ok {
				log.Println("not found: dst:", dst)
				continue
			}

			_, err = crw.Write(b[:n])
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

		crw, err := NewCryptoReadWriter(conn, key)
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			defer func() { _ = conn.Close() }()

			var src [4]byte
			for {
				b := make([]byte, 65535)
				n, err := crw.Read(b)
				if err != nil {
					log.Println(err)
					break
				}

				copy(src[:], b[12:16])

				mtx.RLock()
				stored, ok := crws[src]
				mtx.RUnlock()

				if !ok || stored != crw {
					mtx.Lock()
					crws[src] = crw
					mtx.Unlock()
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
