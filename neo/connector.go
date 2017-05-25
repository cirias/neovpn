package main

import (
	"fmt"
	"io"
	"sync"
)

const BufferSize = 65535

type connector struct {
	lhs  io.ReadWriter
	rhs  io.ReadWriter
	done chan struct{}
}

func newConnector(lhs, rhs io.ReadWriter) *connector {
	return &connector{
		lhs:  lhs,
		rhs:  rhs,
		done: make(chan struct{}),
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
		b := make([]byte, BufferSize)

		for {
			select {
			case <-c.done:
				return
			default:
			}

			n, err := c.lhs.Read(b)
			if err != nil {
				errCh <- fmt.Errorf("could not read from lhs: %v", err)
			}

			_, err = c.rhs.Write(b[:n])
			if err != nil {
				errCh <- fmt.Errorf("could not write to rhs: %v", err)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		b := make([]byte, BufferSize)

		for {
			select {
			case <-c.done:
				return
			default:
			}

			n, err := c.rhs.Read(b)
			if err != nil {
				errCh <- fmt.Errorf("could not read from rhs: %v", err)
			}

			_, err = c.lhs.Write(b[:n])
			if err != nil {
				errCh <- fmt.Errorf("could not write to lhs: %v", err)
			}
		}
	}()

	return errCh
}
