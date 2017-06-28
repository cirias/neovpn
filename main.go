package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var key, sAddr, lAddr, ipAddr string
	flag.StringVar(&key, "key", "", "pre-shared key")
	flag.StringVar(&sAddr, "server", "", "server address for client mode")
	flag.StringVar(&lAddr, "listen", "", "listen address for server mode")
	flag.StringVar(&ipAddr, "ip", "", "ip address in CIDR for local interface")
	flag.Parse()

	if sAddr != "" {
		c, err := newClient(key, sAddr, ipAddr)
		if err != nil {
			log.Fatalln(err)
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case err := <-c.errCh:
			log.Println("exiting on:", err)
			_ = c.Close()
		case sig := <-sigCh:
			log.Println("exiting on:", sig)
			_ = c.Close()
		}
	} else if lAddr != "" {
		/*
		 * if err := server(key, lAddr, ipAddr); err != nil {
		 *   log.Fatalln(err)
		 * }
		 */
	}
}
