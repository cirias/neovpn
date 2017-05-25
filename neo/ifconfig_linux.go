// +build linux

package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

const ipPath = "/bin/ip"

func execute(name string, arg ...string) error {
	if err := exec.Command(name, arg...).Run(); err != nil {
		return fmt.Errorf("could not execute `%v %v`: %v", name, strings.Join(arg, " "), err)
	}

	return nil
}

func ifUp(ifName, ipAddr string) error {
	ip, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return err
	}

	if err := execute(ipPath, "addr", "flush", "dev", ifName); err != nil {
		return err
	}

	if err := execute(ipPath, "addr", "add", ip.String(), "dev", ifName); err != nil {
		return err
	}

	if err := execute(ipPath, "link", "set", "dev", ifName, "up"); err != nil {
		return err
	}

	if err := execute(ipPath, "route", "add", ipNet.String(), "dev", ifName); err != nil {
		return err
	}

	return nil
}

func ifDown(ifName, ipAddr string) error {
	_, ipNet, err := net.ParseCIDR(ipAddr)
	if err != nil {
		return err
	}

	if err := execute(ipPath, "route", "del", ipNet.String(), "dev", ifName); err != nil {
		return err
	}

	if err := execute(ipPath, "link", "set", "dev", ifName, "down"); err != nil {
		return err
	}

	if err := execute(ipPath, "addr", "flush", "dev", ifName); err != nil {
		return err
	}

	return nil
}
