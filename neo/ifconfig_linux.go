// +build linux

package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

const ipCommand = "/bin/ip"

type ifOption struct {
	up   func(string) error
	down func(string) error
}

func execute(name string, arg ...string) error {
	if err := exec.Command(name, arg...).Run(); err != nil {
		return fmt.Errorf("could not execute `%v %v`: %v", name, strings.Join(arg, " "), err)
	}

	return nil
}

func up(ifName string, ops ...ifOption) (func() error, error) {
	var down = func() error {
		for i := len(ops); i >= 0; i-- {
			if err := ops[i].down(ifName); err != nil {
				return err
			}
		}

		if err := execute(ipCommand, "link", "set", "dev", ifName, "up"); err != nil {
			return err
		}
		return nil
	}

	if err := execute(ipCommand, "link", "set", "dev", ifName, "up"); err != nil {
		return down, err
	}

	for _, op := range ops {
		if err := op.up(ifName); err != nil {
			return down, err
		}
	}

	return down, nil
}

func addIP(ipAddr string) ifOption {
	var up = func(ifName string) error {
		return execute(ipCommand, "addr", "add", ipAddr, "dev", ifName)
	}

	var down = func(ifName string) error {
		return execute(ipCommand, "addr", "flush", "dev", ifName)
	}

	return ifOption{up, down}
}

func addRoute(ipNet *net.IPNet, gw net.IP) ifOption {
	var up = func(ifName string) error {
		if gw != nil {
			return execute(ipCommand, "route", "add", ipNet.String(), "via", gw.String(), "dev", ifName)
		}

		return execute(ipCommand, "route", "add", ipNet.String(), "dev", ifName)
	}

	var down = func(ifName string) error {
		if gw != nil {
			return execute(ipCommand, "route", "del", ipNet.String(), "via", gw.String(), "dev", ifName)
		}

		return execute(ipCommand, "route", "del", ipNet.String(), "dev", ifName)
	}

	return ifOption{up, down}
}
