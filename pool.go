package spacemesh

import (
	"context"
	"sync"
	"time"
)

type connectionPool struct {
	connectingMx sync.RWMutex
	connecting   map[int32]chan struct{}

	// initiate a new, not opened connection
	newConnection func() Connection

	cacheMx sync.RWMutex
	cache   map[int32]Connection
}

func newConnectionPool() *connectionPool {
	nc := func() Connection {
		return &connection{
			openingDelay: 1 * time.Second,
		}
	}

	return &connectionPool{
		connecting:    make(map[int32]chan struct{}),
		cache:         make(map[int32]Connection),
		newConnection: nc,
	}
}

func (p *connectionPool) getConnection(ipAddress int32) Connection {
	// cached -> return it immediately
	if conn, ok := p.readFromCache(ipAddress); ok {
		return conn
	}

	// we have 1 channel per ipAddress. If this channel is open, it means we are
	// currently trying to open a connection with this peer.
	p.connectingMx.Lock()
	c, ok := p.connecting[ipAddress]
	p.connectingMx.Unlock()

	if ok {
		<-c

		conn, _ := p.readFromCache(ipAddress)
		return conn
	}

	c = make(chan struct{})
	p.connectingMx.Lock()
	p.connecting[ipAddress] = c
	p.connectingMx.Unlock()

	conn := p.newConnection()
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		conn.Open(ctx)
		close(done)
	}()

	select {
	case <-c: // another goroutine opened the connection for us!
		cancel()     // stop opening the connection
		conn.Close() // will close if it had time to open
	case <-done:
		p.writeToCache(ipAddress, conn)
		close(c)
		return conn
	}

	cachedConnection, _ := p.readFromCache(ipAddress)
	return cachedConnection
}

func (p *connectionPool) onNewRemoteConnection(remotePeer int32, conn Connection) {
	p.connectingMx.Lock()
	defer p.connectingMx.Unlock()

	_, useThisConnection := p.writeToCache(remotePeer, conn)
	if !useThisConnection {
		conn.Close() // blocking
	}

	if c, ok := p.connecting[remotePeer]; ok {
		close(c)
	}
}
