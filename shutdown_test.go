package spacemesh

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestShutdownClearsCache(t *testing.T) {
	t.Parallel()

	a := &testConnection{isOpen: true}
	b := &testConnection{isOpen: true}
	c := &testConnection{isOpen: true}

	p := newConnectionPool()
	p.cache = map[int32]Connection{
		1:     a,
		12321: b,
		999:   c,
	}

	p.shutdown()

	assert.False(t, a.isOpen)
	assert.False(t, b.isOpen)
	assert.False(t, c.isOpen)
	assert.Nil(t, p.cache)
}

func TestShutdownClosesAllConnections(t *testing.T) {
	t.Parallel()

	conns := make([]*testConnection, 0, 10)

	p := newConnectionPool()
	for i := 0; i < 10; i++ {
		conn := &testConnection{isOpen: true}
		conns = append(conns, conn)
		p.cache[int32(i)] = conn
	}

	p.shutdown()

	assert.Len(t, conns, 10)
	for _, conn := range conns {
		assert.False(t, conn.isOpen)
	}
}

func TestShutdownWhileOpeningMany(t *testing.T) {
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
	wg.Add(6)

	for i := 0; i < 6; i++ {
		go func() {
			p.getConnection(7654)
			wg.Done()
		}()
	}

	time.Sleep(50 * time.Millisecond)
	p.shutdown()

	wg.Wait()
	failAfterTimeout(t, &wg, 3*time.Second)

	// we canceled opening every connection
	for _, conn := range conns {
		assert.Equal(t, "opening", conn.id)
		assert.False(t, conn.isOpen)
	}

	assert.Nil(t, p.cache)
}
