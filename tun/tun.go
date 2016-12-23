package tun

import (
	"errors"
	"net"
	"os"
	"os/exec"
)

const MAX_PACKET_SIZE = 2048

type Tun struct {
	file       *os.File
	name       string
	upScript   string
	downScript string
}

func NewTUN(ifName, upScript, downScript string) (*Tun, error) {
	ifce, err := newTUN(ifName)
	if err != nil {
		return nil, err
	}

	ifce.upScript = upScript
	ifce.downScript = downScript

	return ifce, nil
}

func (ifce *Tun) Name() string {
	return ifce.name
}

func (ifce *Tun) Write(p []byte) (n int, err error) {
	n, err = ifce.file.Write(p)
	return
}

func (ifce *Tun) Read() ([]byte, error) {
	for {
		p := make([]byte, MAX_PACKET_SIZE)
		n, err := ifce.file.Read(p)
		if err != nil {
			return p, err
		}

		if n == MAX_PACKET_SIZE {
			return p, errors.New("tun read: max packet size reached")
		}

		// only keep ipv4 packet
		if (p[0] >> 4) == 0x04 {
			return p[:n], nil
		}

		// TODO check packet length
	}
}

func (ifce *Tun) Close() (err error) {
	return ifce.file.Close()
}

func (ifce *Tun) Up(ip net.IP, ipNet *net.IPNet) ([]byte, error) {
	cmd := exec.Command(ifce.upScript, ifce.name, ip.String(), ipNet.String())
	return cmd.CombinedOutput()
}
