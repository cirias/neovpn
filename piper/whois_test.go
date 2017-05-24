package piper

import (
	"context"
	"net"
	"sync"
	"testing"
)

func TestWhoIs(t *testing.T) {
	port := 6543
	ip := net.IPv4(9, 9, 9, 9)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		if context.Canceled != ListenWhoIs(ctx, port, ip) {
			t.Error("ListenWhoIs should quit with Canceled")
		}
	}()

	for {
		if err := BroadcastWhoIs(port, ip); err != nil {
			continue
		} else {
			break
		}
	}

	cancel()
	wg.Wait()
}
