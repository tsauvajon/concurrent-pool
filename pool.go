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

	shuttingDown chan struct{} // when closed, we are shutting down
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
		shuttingDown:  make(chan struct{}),
		newConnection: nc,
	}
}

func (p *connectionPool) getConnection(ipAddress int32) Connection {
	select {
	case <-p.shuttingDown:
		panic("shutting down, we don't want to open a new connection")
	default:
	}

	// cached -> return it immediately
	if conn, ok := p.readFromCache(ipAddress); ok {
		return conn
	}

	// we have 1 channel per ipAddress. If this channel is open, it means we are
	// currently trying to open a connection with this peer.
	p.connectingMx.Lock()
	c, ok := p.connecting[ipAddress]
	if !ok {
		// nobody else is opening a connection to this peer
		c = make(chan struct{})
		p.connecting[ipAddress] = c
	}
	p.connectingMx.Unlock()

	if ok {
		// another goroutine is already creating a connection to this peer
		select {
		case <-p.shuttingDown:
			return nil
		case <-c:
			conn, _ := p.readFromCache(ipAddress)
			return conn
		}
	}

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
		conn.Close() // close if it had time to open anyway
	case <-p.shuttingDown:
		cancel()
		conn.Close()
	case <-done:
		p.writeToCache(ipAddress, conn)
		close(c)
		return conn
	}

	cachedConnection, _ := p.readFromCache(ipAddress)
	return cachedConnection
}

func (p *connectionPool) onNewRemoteConnection(remotePeer int32, conn Connection) {
	select {
	case <-p.shuttingDown:
		conn.Close()
		panic("shutting down, we don't want accept new connections")
	default:
	}

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
