package router

import (
	"encoding/binary"
	"net"
	"sync"
)

type IPPool struct {
	base  net.IP
	count uint32
	used  map[uint32]bool
	mutex sync.Mutex
}

// IPAdd returns a copy of start + add.
// IPAdd(net.IP{192,168,1,1},30) returns net.IP{192.168.1.31}
func IPAdd(start net.IP, add uint32) net.IP { // IPv4 only
	start = start.To4()
	//v := Uvarint([]byte(start))
	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, binary.BigEndian.Uint32(start)+uint32(add))
	//PutUint([]byte(result), v+uint64(add))
	return result
}

func NewIPPool(base net.IP, count uint32) *IPPool {
	return &IPPool{
		base:  base,
		count: count,
		used:  make(map[uint32]bool),
	}
}

func (p *IPPool) Get() net.IP {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i := uint32(1); i <= p.count; i++ {
		if _, ok := p.used[i]; !ok {
			p.used[i] = true
			return IPAdd(p.base, i)
		}
	}

	return nil
}

func (p *IPPool) Put(ip net.IP) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	i := binary.BigEndian.Uint32(ip) - binary.BigEndian.Uint32(p.base)
	p.used[i] = false
}
