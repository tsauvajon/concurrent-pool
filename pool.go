package spacemesh

import (
	"sync"
)

type connectionPool struct {
	connectingQueue chan struct{}
	connecting      map[int32]chan struct{}

	cacheMx sync.RWMutex
	cache   map[int32]Connection
}

func newConnectionPool() *connectionPool {
	return &connectionPool{
		connectingQueue: make(chan struct{}, 1),
		connecting:      make(map[int32]chan struct{}),
		cache:           make(map[int32]Connection),
	}
}

func (p *connectionPool) getConnection(ipAddress int32) Connection {
	// cached -> use it immediately
	if conn, ok := p.cache[ipAddress]; ok {
		return conn
	}

	// only 1 access at a time
	p.connectingQueue <- struct{}{}
	c, ok := p.connecting[ipAddress]

	// we already have a connection opening right now. Wait for it
	// to finish (or have a new remote connection from this peer)
	if ok {
		defer func() { <-c }()
		p.cacheMx.Lock()
		defer p.cacheMx.Unlock()
		return p.cache[ipAddress]
	}

	c = make(chan struct{}, 1)
	// make sure we don't have another goroutine trying to open this
	p.connecting[ipAddress] = c
	<-p.connectingQueue

	conn := &connection{}

	done := make(chan struct{}, 1)
	go func() {
		conn.Open()
		done <- struct{}{}
		close(done)
	}()

	select {
	// case timeout?
	case <-c: // another routine opened the connection for us!
	case <-done:
		p.storeToCache(ipAddress, conn)
		c <- struct{}{}
		close(c)
		return conn
	}

	return p.cache[ipAddress]
}

func (p *connectionPool) onNewRemoteConnection(remotePeer int32, conn Connection) {
	p.connectingQueue <- struct{}{}
	defer func() {<-p.connectingQueue}()
	c, ok := p.connecting[remotePeer]


	p.storeToCache(remotePeer, conn)

	if ok {
		c <- struct{}{}
		close(c)
	}
}
