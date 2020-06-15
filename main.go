package spacemesh

import (
	"sync"
	"time"
)

type Connection interface {
	Open()
	Close()
}

type connection struct {

}

var openConnectionDelay = 1*time.Second

func (c *connection) Open()  {
	time.Sleep(openConnectionDelay)
}
func (c *connection) Close() {}

type connectionPool struct {
	connecting map[int32]chan struct {}

	cacheMx sync.RWMutex
	cache   map[int32]Connection
}

func newConnectionPool() connectionPool {
	return connectionPool{
		connecting: make(map[int32]chan struct {}),
		cache: make(map[int32]Connection),
	}
}


func (p *connectionPool) getConnection(ipAddress int32) Connection {
	c, ok := p.connecting[ipAddress]
	if ok {
		// we already have a connection opening right now. Either wait for it
		// to finish (or have a new remote connection from this peer)
		<-c
		return p.cache[ipAddress]
	}

	c = make(chan struct{})
	// make sure we don't have another goroutine trying to open this
	p.connecting[ipAddress] = c

	conn := &connection{}

	done := make(chan struct{})
	go func () {
		conn.Open()
		done<-struct{}{}
	}()

	select {
	 	case <-c:
		case <-done:
	}

	p.storeToCache(ipAddress, conn)
	c<-struct{}{}

	return conn
}

func (p *connectionPool) onNewRemoteConnection(remotePeer int32, conn Connection) {
	c, ok := p.connecting[remotePeer]
	if ok {
		p.storeToCache(remotePeer, conn)
		c<-struct{}{}
	}
}
