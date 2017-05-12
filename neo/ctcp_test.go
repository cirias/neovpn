package neo

import (
	"fmt"
	"sync"
	"testing"
)

func TestReadWrite(t *testing.T) {
	ln, err := Listen("psk", "127.0.0.1:45645")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := Dial("psk", "127.0.0.1:45645")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := make([]byte, 1024)
		n, err := conn.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if fmt.Sprintf("%s", b[:n]) != "hello world" {
			t.Error("received message should be same as sent")
		}
	}()

	{
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := []byte("hello world")
		n, err := conn.Write(b)
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
	ln, err := Listen("psk", "127.0.0.1:45645")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := Dial("psk", "127.0.0.1:45645")
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		b := []byte("hello world")
		n, err := conn.Write(b)
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
		defer conn.Close()

		b := make([]byte, 1024)
		n, err := conn.Read(b)
		if err != nil {
			t.Fatal(err)
		}

		if fmt.Sprintf("%s", b[:n]) != "hello world" {
			t.Error("received message should be same as sent")
		}
	}

	wg.Wait()
}
