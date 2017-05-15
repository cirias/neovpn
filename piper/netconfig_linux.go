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

	if err := exec.Command("ip", "link", "set", "dev", ifname, "up").Run(); err != nil {
		return err
	}
}
