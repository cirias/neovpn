package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type server struct {
	mtx   sync.RWMutex
	conns map[[4]byte]*stoppableConn
	tun   io.Closer
	ln    io.Closer
	down  func() error
}

func newServer(key, lAddr, ipAddr string) (*server, error) {
	tun, err := newTun()
	if err != nil {
		return nil, fmt.Errorf("could not new tun: %v", err)
	}

	down, err := up(tun.Name(), addIP(ipAddr))
	if err != nil {
		return nil, fmt.Errorf("could not turn interface up: %v", err)
	}

	ln, err := net.Listen("tcp", lAddr)
	if err != nil {
		return nil, fmt.Errorf("could not listen to %v@%v: %v", key, lAddr, err)
	}
	log.Printf("listening to %v@%v\n", key, lAddr)

	conns := make(map[[4]byte]*stoppableConn)
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
				for _, c := range conns {
					if _, err := c.Write(b[:n]); err != nil {
						log.Println(err)
					}
				}
			}

			mtx.RLock()
			c, ok := conns[dst]
			mtx.RUnlock()
			if !ok {
				log.Println("not found: dst:", dst)
				continue
			}

			_, err = c.Write(b[:n])
			if err != nil {
				// TODO
				// errCh <- fmt.Errorf("could not write to %v: %v\n", conn.RemoteAddr(), err)
				errCh <- fmt.Errorf("could not write to connection: %v", err)

				mtx.Lock()
				if stored, ok := conns[dst]; ok && stored == c {
					delete(conns, dst)
				}
				mtx.Unlock()
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- fmt.Errorf("could not accept connection: %v", err)
				continue
			}

			go func(conn net.Conn) {
				defer func() { _ = conn.Close() }()

				cc, err := newCryptoConn(conn, key)
				if err != nil {
					errCh <- fmt.Errorf("could not new crypto connection: %v", err)
					return
				}

				sc := newStoppableConn(cc)

				/*
				 * defer func() {
				 *   mtx.Lock()
				 *   if stored, ok := conns[src]; ok && stored == cc {
				 *     delete(conns, src)
				 *   }
				 *   mtx.Unlock()
				 * }()
				 */

				var src [4]byte
				for {
					b := make([]byte, 65535)

					if err := sc.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
						errCh <- fmt.Errorf("could not set read deadline: %v", err)
						break
					}

					n, err := sc.Read(b)
					if err != nil {
						errCh <- fmt.Errorf("could not read from %v: %v", sc.RemoteAddr(), err)
						break
					}

					copy(src[:], b[12:16])

					mtx.RLock()
					stored, ok := conns[src]
					mtx.RUnlock()

					if !ok || stored != sc {
						mtx.Lock()
						conns[src] = sc
						mtx.Unlock()
					}

					_, err = tun.Write(b[:n])
					if err != nil {
						errCh <- fmt.Errorf("could not write to tun: %v", err)
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

	return &server{
		mtx:   mtx,
		conns: conns,
		tun:   tun,
		ln:    ln,
		down:  down,
	}, nil
}

func (s *server) Close() error {
	if err := s.down(); err != nil {
		return err
	}

	if err := s.ln.Close(); err != nil {
		return err
	}

	s.mtx.RLock()
	for _, c := range s.conns {
		c.stop()
	}
	s.mtx.RUnlock()

	if err := s.tun.Close(); err != nil {
		return err
	}

	return nil
}
