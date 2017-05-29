package main

import (
	"net"
	"sync"
	"testing"
)

func TestPipe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		c0, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}

		c1, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}

		if err := <-pipe(c0, c1); err != nil {
			t.Fatal(err)
		}
	}()

	c0, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	c1, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c0.Write([]byte("i am c0"))
		if err != nil {
			t.Fatal(err)
		}

		b := make([]byte, 1024)
		n, err := c0.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if string(b[:n]) != "i am c1" {
			t.Error("c0 should receive c1")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c1.Write([]byte("i am c1"))
		if err != nil {
			t.Fatal(err)
		}

		b := make([]byte, 1024)
		n, err := c1.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if string(b[:n]) != "i am c0" {
			t.Error("c1 should receive c0")
		}
	}()
}
