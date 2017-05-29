package main

import (
	"fmt"
	"io"
	"log"
	"sync"
)

func pipe(lhs, rhs io.ReadWriter) <-chan error {
	var wg sync.WaitGroup

	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("copy to rhs from lhs")
		_, err := io.Copy(rhs, lhs)
		if err != nil {
			errCh <- fmt.Errorf("could not copy to rhs from lhs: %v", err)
		}
		log.Println("copy to rhs from lhs done")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("copy to lhs from rhs")
		_, err := io.Copy(lhs, rhs)
		if err != nil {
			errCh <- fmt.Errorf("could not copy to lhs from rhs: %v", err)
		}
		log.Println("copy to lhs from rhs done")
	}()

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return errCh
}
