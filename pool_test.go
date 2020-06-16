package spacemesh

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testConnection struct {
	openingDelay time.Duration
	isOpen       bool
	id           string
}

func (c *testConnection) Open(ctx context.Context) {
	select {
	case <-time.After(c.openingDelay):
		c.isOpen = true
	case <-ctx.Done():
	}
}

func (c *testConnection) Close() { c.isOpen = false }

func failAfterTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Error("timed out")
		t.FailNow()
	}
}

func TestNewConnectionPool(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	conn := p.newConnection()

	got, ok := conn.(*connection)
	assert.True(t, ok)
	assert.False(t, got.isOpen)
	assert.Equal(t, 1*time.Second, got.openingDelay)
}

func TestOpenConcurrently(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.newConnection = func() Connection {
		return &connection{
			openingDelay: 200 * time.Millisecond,
			isOpen:       false,
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(8)

	for i := 0; i < 8; i++ {
		go func() {
			p.getConnection(234)
			wg.Done()
		}()
	}

	failAfterTimeout(t, &wg, 500*time.Millisecond)

	conn, ok := p.cache[234]
	assert.True(t, ok)

	got, ok := conn.(*connection)
	assert.True(t, ok)
	assert.True(t, got.isOpen)
}

func TestIncomingWhileOpening(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.newConnection = func() Connection {
		return &connection{
			openingDelay: 1 * time.Second,
		}
	}
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		p.getConnection(234)
		wg.Done()
	}()
	go func() {
		p.onNewRemoteConnection(234, &testConnection{id: "remote"})
		wg.Done()
	}()

	failAfterTimeout(t, &wg, 2*time.Second)

	conn, ok := p.cache[234]
	assert.True(t, ok)

	got, ok := conn.(*testConnection)
	assert.True(t, ok)
	assert.Equal(t, "remote", got.id)
}

func TestIncomingWhileOpeningMany(t *testing.T) {
	t.Parallel()

	mx := sync.RWMutex{}
	conns := make([]*testConnection, 0)

	p := newConnectionPool()
	p.newConnection = func() Connection {
		conn := &testConnection{
			openingDelay: 2 * time.Second,
			id:           "opening",
		}

		mx.Lock()
		defer mx.Unlock()
		conns = append(conns, conn)

		return conn
	}

	wg := sync.WaitGroup{}
	wg.Add(7)

	for i := 0; i < 6; i++ {
		go func() {
			p.getConnection(7654)
			wg.Done()
		}()
	}

	time.Sleep(50 * time.Millisecond)

	go func() {
		p.onNewRemoteConnection(7654, &testConnection{
			isOpen: true,
			id:     "remote",
		})
		wg.Done()
	}()

	failAfterTimeout(t, &wg, 3*time.Second)

	// we only tried to open 1 connection
	assert.Len(t, conns, 1)

	// we canceled opening every connection
	for _, conn := range conns {
		assert.Equal(t, "opening", conn.id)
		assert.False(t, conn.isOpen)
	}

	// the cached connection is the one opened by the remote peer
	conn, ok := p.readFromCache(7654)
	assert.True(t, ok)

	got, ok := conn.(*testConnection)
	assert.True(t, ok)

	assert.True(t, got.isOpen)
	assert.Equal(t, "remote", got.id)
}

func TestUseCachedConn(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.newConnection = func() Connection {
		return &testConnection{
			openingDelay: 200 * time.Millisecond,
			isOpen:       false,
		}
	}

	done := make(chan struct{})

	go func() {
		p.getConnection(234)
		p.getConnection(234)
		close(done)
	}()

	select {
	case <-done:
		return
	case <-time.After(1 * time.Second):
		t.Fail()
	}
}

// - we have a cached connection for peer 4
// - onNewConnection is called by peer 4
// => we close that connection
func TestClosesNewRemoteConnIfNotUsed(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.cache[4] = &connection{isOpen: true}

	conn := &connection{isOpen: true}
	p.onNewRemoteConnection(4, conn)

	assert.False(t, conn.isOpen)
}

// Same situation as last test but different hosts => not closing
func TestClosesNewRemoteConnIfNotUsedOnlySameHost(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.cache[4] = &connection{isOpen: true}

	conn := &connection{isOpen: true}
	p.onNewRemoteConnection(88989898, conn)

	assert.True(t, conn.isOpen)

	_, ok := p.cache[88989898]
	assert.True(t, ok)
}

// if we're trying to open a connection with a peer and we receive a remote
// connection from this peer, we should stop trying to open it.
func TestClosesNewConnIfNotUsed(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	conn := &connection{
		openingDelay: 1 * time.Second,
	}
	p.newConnection = func() Connection { return conn }

	done := make(chan struct{})
	go func() {
		p.getConnection(56)
		close(done)
	}()

	p.onNewRemoteConnection(56, &connection{isOpen: true})

	select {
	case <-done:
		assert.False(t, conn.isOpen)
	case <-time.After(2 * time.Second):
		t.Fail()
	}
}

func TestClosesNewConnIfNotUsedOnlySameHost(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	conn := &connection{
		openingDelay: 200 * time.Millisecond,
	}
	p.newConnection = func() Connection { return conn }

	done := make(chan struct{})
	go func() {
		p.getConnection(56)
		close(done)
	}()

	p.onNewRemoteConnection(121212121, &connection{isOpen: true})

	select {
	case <-done:
		assert.True(t, conn.isOpen)
	case <-time.After(1 * time.Second):
		t.Fail()
	}
}

func TestGetConnectionDuringShutdown(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	close(p.shuttingDown)

	assert.PanicsWithValue(
		t,
		"shutting down, we don't want to open a new connection",
		func() {
			p.getConnection(123123)
		},
	)
}

func TestOnNewRemoteConnectionDuringShutdown(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	close(p.shuttingDown)

	conn := &testConnection{isOpen: true}

	assert.PanicsWithValue(
		t,
		"shutting down, we don't want accept new connections",
		func() {
			p.onNewRemoteConnection(99, conn)
		},
	)

	assert.False(t, conn.isOpen)
}
