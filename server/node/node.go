package node

import "github.com/cirias/neovpn/tunnel"

type Node struct {
	tunnel.Conn
}

func (n *Node) AssignAddr() {
}
