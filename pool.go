// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tunnel

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"golang.org/x/net/http2"

	"github.com/myENA/go-http-tunnel/id"
)

type OnDisconnectListener func(identifier id.ID)

type connPair struct {
	conn       net.Conn
	clientConn *http2.ClientConn
}

type ConnPool struct {
	t        *http2.Transport
	conns    map[string]connPair // key is host:port
	listener OnDisconnectListener
	mu       sync.RWMutex
}

func newConnPool(t *http2.Transport, l OnDisconnectListener) *ConnPool {
	return &ConnPool{
		t:        t,
		listener: l,
		conns:    make(map[string]connPair),
	}
}

func (p *ConnPool) GetClientConn(req *http.Request, addr string) (*http2.ClientConn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if cp, ok := p.conns[addr]; ok && cp.clientConn.CanTakeNewRequest() {
		return cp.clientConn, nil
	}

	return nil, errClientNotConnected
}

func (p *ConnPool) MarkDead(c *http2.ClientConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, cp := range p.conns {
		if cp.clientConn == c {
			cp.conn.Close()
			delete(p.conns, addr)
			if p.listener != nil {
				p.listener(p.addrToIdentifier(addr))
			}
			return
		}
	}
}

func (p *ConnPool) AddConn(conn net.Conn, identifier id.ID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	addr := p.addr(identifier)

	if _, ok := p.conns[addr]; ok {
		return errClientAlreadyConnected
	}

	c, err := p.t.NewClientConn(conn)
	if err != nil {
		return err
	}
	p.conns[addr] = connPair{
		conn:       conn,
		clientConn: c,
	}

	return nil
}

func (p *ConnPool) DeleteConn(identifier id.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	addr := p.addr(identifier)

	if cp, ok := p.conns[addr]; ok {
		cp.conn.Close()
		delete(p.conns, addr)
		if p.listener != nil {
			p.listener(identifier)
		}
	}
}

func (p *ConnPool) URL(identifier id.ID) string {
	return fmt.Sprint("https://", identifier)
}

func (p *ConnPool) addr(identifier id.ID) string {
	return fmt.Sprint(identifier.String(), ":443")
}

func (p *ConnPool) addrToIdentifier(addr string) id.ID {
	identifier := id.ID{}
	identifier.UnmarshalText([]byte(addr[:len(addr)-4]))
	return identifier
}
