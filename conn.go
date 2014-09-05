package tcpez

import (
	"math/rand"
	"net"
	"sync"
	"time"
)

type ConnectionPool struct {
	Addresses []string
	Initial   int
	Timeout   time.Duration
	conns     []net.Conn
	sync.Mutex
}

func NewConnectionPool(addresses []string, initial int, timeout time.Duration) (p *ConnectionPool, err error) {
	p = &ConnectionPool{Addresses: addresses, Initial: initial, Timeout: timeout}
	errs := make([]error, 0)
	for i := 0; i < initial; i++ {
		conn, err := p.dial()
		if err != nil {
			errs = append(errs, err)
		} else {
			p.conns = append(p.conns, conn)
		}
	}
	// Only errors, no real connections
	if len(p.conns) == 0 {
		return nil, errs[0]
	}
	return p, nil
}

func (p *ConnectionPool) Take() (c net.Conn, err error) {
	p.Lock()
	defer p.Unlock()
	if len(p.conns) > 0 {
		// shift a conn off the array
		c = p.conns[0]
		p.conns = p.conns[1:len(p.conns)]
		return c, nil
	} else {
		return p.dial()
	}
}

func (p *ConnectionPool) Return(c net.Conn) {
	p.Lock()
	defer p.Unlock()
	p.conns = append(p.conns, c)
}

func (p *ConnectionPool) dial() (c net.Conn, err error) {
	address := p.Addresses[rand.Intn(len(p.Addresses))]
	log.Debug("Dial address %s", address)
	return net.DialTimeout("tcp", address, p.Timeout)
}
