package spacemesh

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadFromCache(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.cache[999] = &testConnection{
		id: "12",
	}

	c, ok := p.readFromCache(999)
	assert.True(t, ok)

	got, ok := c.(*testConnection)
	assert.True(t, ok)
	assert.Equal(t, "12", got.id)

	_, ok = p.readFromCache(1)
	assert.False(t, ok)
}

func TestWriteToCache(t *testing.T) {
	t.Parallel()

	var (
		a = &testConnection{id: "a"}
		b = &testConnection{id: "b"}
		c = &testConnection{id: "c"}
	)

	p := newConnectionPool()
	wg := sync.WaitGroup{}
	wg.Add(2)

	p.writeToCache(1, a)

	go func() {
		p.writeToCache(1, b)
		wg.Done()
	}()
	go func() {
		p.writeToCache(12312312, c)
		wg.Done()
	}()

	failAfterTimeout(t, &wg, 500*time.Millisecond)

	conn, ok := p.cache[1]
	assert.True(t, ok)

	got, ok := conn.(*testConnection)
	assert.True(t, ok)
	assert.Equal(t, "a", got.id)

	conn, ok = p.cache[12312312]
	assert.True(t, ok)

	got, ok = conn.(*testConnection)
	assert.True(t, ok)
	assert.Equal(t, "c", got.id)
}

func TestPoolSetsCache(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.getConnection(234)

	c, ok := p.cache[234]
	assert.True(t, ok)
	assert.NotNil(t, c)
}

func TestPoolReadsCache(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.cache[999] = &testConnection{id: "cached"}
	p.newConnection = func() Connection {
		return &testConnection{id: "new"}
	}

	done := make(chan struct{})
	go func() {
		p.getConnection(999)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fail()
		return
	}

	conn := p.cache[999]
	got, ok := conn.(*testConnection)
	assert.True(t, ok)
	assert.Equal(t, "cached", got.id)
}
