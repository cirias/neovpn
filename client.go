package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

type client struct {
	conn  io.Closer
	tun   io.Closer
	down  func() error
	errCh chan error
}

func newClient(key, rAddr, ipAddr string) (*client, error) {
	conn, err := net.Dial("tcp", rAddr)
	if err != nil {
		return nil, fmt.Errorf("could not dial to %v@%v: %v", key, rAddr, err)
	}

	crw, err := newCryptoConn(conn, key)
	if err != nil {
		return nil, fmt.Errorf("could not new CryptoReadWriter: %v", err)
	}

	tun, err := newTun()
	if err != nil {
		return nil, fmt.Errorf("could not new tun: %v", err)
	}

	down, err := up(tun.Name(), addIP(ipAddr))
	if err != nil {
		return nil, fmt.Errorf("could not turn interface up: %v", err)
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)

		for err := range pipe(tun, crw) {
			errCh <- fmt.Errorf("could not pipe tun and crw: %v\n", err)
		}
	}()

	return &client{
		conn:  conn,
		tun:   tun,
		down:  down,
		errCh: errCh,
	}, nil
}

func (c *client) Close() error {
	if err := c.down(); err != nil {
		log.Printf("could not turn interface down: %v\n", err)
	}

	if err := c.tun.Close(); err != nil {
		return fmt.Errorf("could not close tun: %v\n", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("could not close conn: %v\n", err)
	}

	for _ = range c.errCh {

	}

	return nil
}
