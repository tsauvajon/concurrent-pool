package spacemesh

import (
	"sync"
)

type connectionPool struct {
	connectingMx sync.RWMutex
	connecting map[int32]chan struct{}

	cacheMx sync.RWMutex
	cache   map[int32]Connection
}

func newConnectionPool() *connectionPool {
	return &connectionPool{
		connecting:      make(map[int32]chan struct{}, 0),
		cache:           make(map[int32]Connection),
	}
}

func (p *connectionPool) getConnection(ipAddress int32) Connection {
	// cached -> return it immediately
	if conn, ok := p.getFromCache(ipAddress); ok {
		return conn
	}

	p.connectingMx.Lock()
	c, ok := p.connecting[ipAddress]

	// we already have a connection opening right now. Wait for it
	// to finish (or have a new remote connection from this peer)
	if ok {
		<-c
		p.connectingMx.Unlock()

		p.cacheMx.Lock()
		defer p.cacheMx.Unlock()

		return p.cache[ipAddress]
	}

	c = make(chan struct{}, 0)
	p.connecting[ipAddress] = c
	p.connectingMx.Unlock()

	conn := &connection{}

	done := make(chan struct{}, 0)
	go func() {
		conn.Open()
		close(done)
	}()

	select {
	case <-c: // another goroutine opened the connection for us!
		go func() {
			// I'm assuming we don't have a wait to cancel opening the
			// connection.
			// In that case, we have to wait for the connection to open,
			// and we close it immediately.
			<-done
			conn.Close()
		}()
	case <-done:
		p.storeToCache(ipAddress, conn)
		close(c)
		return conn
	}

	cachedConnection, _ := p.getFromCache(ipAddress)
	return cachedConnection
}

func (p *connectionPool) onNewRemoteConnection(remotePeer int32, conn Connection) {
	p.connectingMx.Lock()
	defer p.connectingMx.Unlock()

	p.storeToCache(remotePeer, conn)

	c, ok := p.connecting[remotePeer]
	if ok {
		close(c)
	}
}
