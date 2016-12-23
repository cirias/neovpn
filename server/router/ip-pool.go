package router

import (
	"encoding/binary"
	"net"
	"sync"
)

var (
	s uint32 = 256
	m uint32 = 256 * 256
	l uint32 = 256 * 256 * 256
)

type IPPool struct {
	start uint32
	end   uint32
	used  map[uint32]bool
	mutex sync.Mutex
}

func NewIPPool(ip net.IP, ipNet *net.IPNet) *IPPool {
	start := binary.BigEndian.Uint32(ipNet.IP)
	ones, bits := ipNet.Mask.Size()
	end := start + 2<<uint(bits-ones-1)

	used := make(map[uint32]bool)
	used[binary.BigEndian.Uint32(ip.To4())] = true

	return &IPPool{
		start: start,
		end:   end,
		used:  used,
	}
}

func (p *IPPool) Get() net.IP {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i := p.start; i <= p.end; i++ {
		if _, ok := p.used[i]; ok {
			continue
		}

		if (i%s) == 0 ||
			(i%m) == 0 ||
			(i%l) == 0 {
			continue
		}

		p.used[i] = true
		result := make(net.IP, 4)
		binary.BigEndian.PutUint32(result, i)
		return result
	}

	return nil
}

func (p *IPPool) Put(ip net.IP) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	i := binary.BigEndian.Uint32(ip)
	p.used[i] = false
}
