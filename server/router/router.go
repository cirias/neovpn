package router

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/cirias/neovpn/tun"
	"github.com/cirias/neovpn/tunnel"
)

type Node struct {
	*tunnel.Conn
	IP [4]byte
}

type Router struct {
	nodesMutex sync.RWMutex
	ipsMutex   sync.RWMutex
	tun        *tun.Tun
	ipPool     *IPPool
	nodes      map[[4]byte]*Node
	ips        map[string]net.IP // map[id]ip
}

func NewRouter(ip net.IP, count uint32, tun *tun.Tun) *Router {
	r := &Router{
		tun:    tun,
		ipPool: NewIPPool(ip, count),
		nodes:  make(map[[4]byte]*Node),
		ips:    make(map[string]net.IP),
	}

	go func() {
		for {
			ipPacket, err := tun.Read()
			if err != nil {
				log.Println(err)
				continue
			}

			if err := r.handleIPPacketFromTun(ipPacket); err != nil {
				log.Println(err)
			}
		}
	}()

	return r
}

func (r *Router) Take(c *tunnel.Conn) {
	n := &Node{Conn: c}
	defer n.Close()
	defer func() {
		r.nodesMutex.Lock()
		delete(r.nodes, n.IP)
		r.nodesMutex.Unlock()

		r.ipPool.Put(n.IP[:])
	}()

RECEIVE_LOOP:
	for {
		pack, err := n.Receive()
		if err != nil {
			log.Println(err)
			break
		}

		switch pack.Header.Type {
		case tunnel.IP_REQUEST:
			if err := r.handleIPRequest(n, pack.Payload); err != nil {
				log.Println(err)
				break RECEIVE_LOOP
			}
		case tunnel.IP_PACKET:
			if err := r.handleIPPacket(n, pack.Payload); err != nil {
				log.Println(err)
				break RECEIVE_LOOP
			}
		default:
			log.Println("invalid type")
			break RECEIVE_LOOP
		}
	}
}

func (r *Router) handleIPRequest(n *Node, id []byte) error {
	log.Println("receive id", id)
	if n.IP != [4]byte{} {
		return errors.New("already has an IP: " + fmt.Sprint(n.IP))
	}

	r.ipsMutex.RLock()
	ip, ok := r.ips[string(id)]
	r.ipsMutex.RUnlock()
	if !ok {
		ip = r.ipPool.Get()

		r.ipsMutex.RLock()
		r.ips[string(id)] = ip
		r.ipsMutex.RUnlock()
	}

	var ip4 [4]byte
	copy(ip4[:], ip)
	n.IP = ip4

	r.nodesMutex.Lock()
	r.nodes[ip4] = n
	r.nodesMutex.Unlock()

	return n.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_RESPONSE,
		},
		Payload: ip,
	})
}

func (r *Router) handleIPPacket(n *Node, packet []byte) error {
	log.Println("receive packet", packet)
	if _, err := r.tun.Write(packet); err != nil {
		return err
	}

	return nil
}

func (r *Router) handleIPPacketFromTun(packet []byte) error {
	var dst [4]byte
	copy(dst[:], packet[16:20])

	r.nodesMutex.RLock()
	n, ok := r.nodes[dst]
	r.nodesMutex.RUnlock()
	if !ok {
		return errors.New("nodes not found: " + fmt.Sprint(dst))
	}

	return n.Send(&tunnel.Pack{
		Header: &tunnel.Header{
			Type: tunnel.IP_PACKET,
		},
		Payload: packet,
	})
}
