package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type server struct {
	tun  io.Closer
	ln   io.Closer
	down func() error
}

func newServer(key, lAddr, ipAddr string) (*server, error) {
	tun, err := newTun()
	if err != nil {
		return nil, fmt.Errorf("could not new tun: %v", err)
	}
	defer func() { _ = tun.Close() }()

	down, err := up(tun.Name(), addIP(ipAddr))
	if err != nil {
		return nil, fmt.Errorf("could not turn interface up: %v", err)
	}
	defer func() {
		if err := down(); err != nil {
			log.Printf("could not turn interface down: %v\n", err)
		}
	}()

	ln, err := net.Listen("tcp", lAddr)
	if err != nil {
		return nil, fmt.Errorf("could not listen to %v@%v: %v", key, lAddr, err)
	}
	log.Printf("listening to %v@%v", key, lAddr)
	defer func() { _ = ln.Close() }()

	crws := make(map[[4]byte]io.ReadWriter)
	var mtx sync.RWMutex

	var wg sync.WaitGroup
	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- fmt.Errorf("could not accept connection: %v\n", err)
				continue
			}

			go func(conn net.Conn) {
				defer func() { _ = conn.Close() }()

				crw, err := newCryptoReadWriter(conn, key)
				if err != nil {
					errCh <- fmt.Errorf("could not new cryptoReadWriter: %v\n", err)
					return
				}

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
			}(conn)
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return &server{}, nil
}

func (s *server) Close() error {
	return nil
}
