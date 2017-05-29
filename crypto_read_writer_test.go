package main

import (
	"net"
	"sync"
	"testing"
)

const key = "psk"

func TestReadWrite(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()

		rw, err := NewCryptoReadWriter(conn, "psk")
		if err != nil {
			t.Fatal(err)
		}

		b := make([]byte, 1024)
		n, err := rw.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if string(b[:n]) != "hello world" {
			t.Error("received message should be same as sent")
		}
	}()

	{
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()

		rw, err := NewCryptoReadWriter(conn, "psk")
		if err != nil {
			t.Fatal(err)
		}

		b := []byte("hello world")
		n, err := rw.Write(b)
		if err != nil {
			t.Fatal(err)
		}

		if len(b) != n {
			t.Error("written should equal len(b)", n, b)
		}
	}

	wg.Wait()
}

func TestWriteRead(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()

		rw, err := NewCryptoReadWriter(conn, "psk")
		if err != nil {
			t.Fatal(err)
		}

		b := []byte("hello world")
		n, err := rw.Write(b)
		if err != nil {
			t.Fatal(err)
		}

		if len(b) != n {
			t.Error("written should equal len(b)", n, b)
		}
	}()

	{
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()

		rw, err := NewCryptoReadWriter(conn, "psk")
		if err != nil {
			t.Fatal(err)
		}

		b := make([]byte, 1024)
		n, err := rw.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if string(b[:n]) != "hello world" {
			t.Error("received message should be same as sent")
		}
	}

	wg.Wait()
}
