// +build linux

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	IFF_TUN   = 0x0001
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
)

type ifRequest struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

func newTun() (*tuntap, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}

	name, err := createInterface(file.Fd(), "", IFF_TUN|IFF_NO_PI)
	if err != nil {
		return nil, fmt.Errorf("could not create interface: %v", err)
	}

	return &tuntap{
		name: name,
		File: file,
	}, nil
}

func createInterface(fd uintptr, name string, flags uint16) (string, error) {
	var req ifRequest
	req.Flags = flags
	copy(req.Name[:], name)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return "", fmt.Errorf("could call sys_ioctl: %v", errno)
	}

	return strings.Trim(string(req.Name[:]), "\x00"), nil
}
