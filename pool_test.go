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
	ipAddress    int32
}

func (c *testConnection) Open(ctx context.Context) {
	select {
	case <-time.After(c.openingDelay):
	case <-ctx.Done():
	}
}

func (c *testConnection) Close() {}

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
		t.FailNow()
	}
}

func TestOpenConcurrent(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.newConnection = func() Connection {
		return &testConnection{
			openingDelay: 100 * time.Millisecond,
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

	failAfterTimeout(t, &wg, 300*time.Millisecond)

}

func TestIncomingWhileOpening(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		p.getConnection(234)
		wg.Done()
	}()
	go func() {
		p.onNewRemoteConnection(234, &connection{})
		wg.Done()
	}()

	failAfterTimeout(t, &wg, 100*time.Millisecond)
}

func TestUseCachedConn(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.newConnection = func() Connection {
		return &testConnection{
			openingDelay: 100 * time.Millisecond,
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
	case <-time.After(150 * time.Millisecond):
		t.Fail()
	}
}

// if we keep the cached connection in onNewRemoteConnection, we can close this
// new incoming connection.
func TestClosesConnIfNotUsed(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.cache[4] = &connection{isOpen: true}

	conn := &connection{isOpen: true}
	p.onNewRemoteConnection(4, conn)

	assert.False(t, conn.isOpen)
}
