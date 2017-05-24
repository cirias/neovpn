package piper

import (
	"context"
	"net"
	"time"
)

func BroadcastWhoIs(port int, ip net.IP) error {
	raddr := &net.UDPAddr{
		net.IPv4bcast,
		port,
		"",
	}

	// `DialUDP` can't receive data from conn when doing broadcast, use `ListenUDP` instead
	// https://github.com/golang/go/issues/13391
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.WriteToUDP(ip.To4()[:4], raddr)
	if err != nil {
		return err
	}

	b := make([]byte, 32)

	if err = conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		return err
	}

	_, err = conn.Read(b)

	return err
}

func ListenWhoIs(ctx context.Context, port int, ip net.IP) error {
	laddr := &net.UDPAddr{
		net.IPv4zero,
		port,
		"",
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 32)

	for {
		if err := conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}
		n, addr, err := conn.ReadFromUDP(buf)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				continue
			}
		}
		if err != nil {
			return err
		}

		if n != 4 {
			continue
		}
		if !net.IPv4(buf[0], buf[1], buf[2], buf[3]).Equal(ip) {
			continue
		}

		_, err = conn.WriteToUDP([]byte("ack"), addr)
		if err != nil {
			return err
		}
	}
}
