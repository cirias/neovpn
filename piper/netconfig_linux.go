// +build linux

package piper

import (
	"net"
	"os/exec"
)

func IfUp(ifname, cidr string) error {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	if err := exec.Command("ip", "addr", "flush", "dev", ifname).Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "addr", "add", ip.String(), "dev", ifname).Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "link", "set", "dev", ifname, "up").Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "route", "add", ipNet.String(), "dev", ifname).Run(); err != nil {
		return err
	}

	return nil
}

func IfDown(ifname, cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	if err := exec.Command("ip", "route", "del", ipNet.String(), "dev", ifname).Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "link", "set", "dev", ifname, "down").Run(); err != nil {
		return err
	}

	if err := exec.Command("ip", "addr", "flush", "dev", ifname).Run(); err != nil {
		return err
	}

	return nil
}
