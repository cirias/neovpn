package main

import (
	"fmt"
	"io"
	"sync"
)

func pipe(lhs, rhs io.ReadWriter) <-chan error {
	var wg sync.WaitGroup

	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(rhs, lhs)
		if err != nil {
			errCh <- fmt.Errorf("could not copy lhs to rhs: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(rhs, lhs)
		if err != nil {
			errCh <- fmt.Errorf("could not copy lhs to rhs: %v", err)
		}
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return errCh
}
