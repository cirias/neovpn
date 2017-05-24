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

type connector struct {
	first  io.ReadWriter
	second io.ReadWriter
	done   chan struct{}
}

func newConnector(first, second io.ReadWriter) *connector {
	return &connector{
		first:  first,
		second: second,
		done:   make(chan struct{}),
	}
}

func (c *connector) Close() {
	close(c.done)
}

func (c *connector) Run() <-chan error {
	var wg sync.WaitGroup

	errCh := make(chan error)
	defer func() {
		wg.Wait()
		close(errCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		b := make([]byte, 65535)

		for {
			select {
			case <-c.done:
				return
			default:
			}

			n, err := c.first.Read(b)
			if err != nil {
				errCh <- fmt.Errorf("could not read from first: %v", err)
			}

			_, err = c.second.Write(b[:n])
			if err != nil {
				errCh <- fmt.Errorf("could not write to second: %v", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		b := make([]byte, 65535)

		for {
			select {
			case <-c.done:
				return
			default:
			}

			n, err := c.second.Read(b)
			if err != nil {
				errCh <- fmt.Errorf("could not read from second: %v", err)
			}

			_, err = c.first.Write(b[:n])
			if err != nil {
				errCh <- fmt.Errorf("could not write to first: %v", err)
			}
		}
	}()

	return errCh
}

func client(key, address string) {
	tun, err := newTun()
	if err != nil {
		log.Fatalf("could not new first: %v\n", err)
	}
	defer func() { _ = tun.Close() }()

	conn, err := Dial(key, address)
	if err != nil {
		log.Fatalf("could not dial to %v@%v: %v\n", key, address, err)
	}
	defer func() { _ = conn.Close() }()

	c := newConnector(tun, conn)
	defer c.Close()

	for err := range c.Run() {
		log.Fatalf("could not run: %v\n", err)
	}
}

func server(key, address string) {
	tun, err := newTun()
	if err != nil {
		log.Fatalln(err)
	}
	defer func() { _ = tun.Close() }()

	ln, err := Listen(key, address)
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
