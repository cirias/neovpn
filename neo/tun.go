package main

import "os"

type tuntap struct {
	name string
	*os.File
}

func (t *tuntap) Name() string {
	return t.name
}
