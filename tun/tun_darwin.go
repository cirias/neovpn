// +build darwin

package tun

import "os"

func newTUN(ifName string) (ifce *Tun, err error) {
	file, err := os.OpenFile("/dev/"+ifName, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	ifce = &Tun{file: file, name: ifName}
	return
}
