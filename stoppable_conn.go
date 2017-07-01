package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type stoppableConn struct {
	done chan struct{}
	wg   sync.WaitGroup
	net.Conn
}

func newStoppableConn(conn net.Conn) *stoppableConn {
	return &stoppableConn{
		done: make(chan struct{}),
		Conn: conn,
	}
}

func (c *stoppableConn) Read(b []byte) (int, error) {
	c.wg.Add(1)
	defer c.wg.Done()

	for {
		if err := c.Conn.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return 0, fmt.Errorf("could not set read deadline: %v", err)
		}

		n, err := c.Conn.Read(b)
		if e, ok := err.(net.Error); ok && e.Timeout() {
			select {
			case <-c.done:
				return 0, fmt.Errorf("connection has been stopped")
			default:
				continue
			}
		}

		return n, err
	}
}

func (c *stoppableConn) stop() {
	close(c.done)
	c.wg.Wait()
}
