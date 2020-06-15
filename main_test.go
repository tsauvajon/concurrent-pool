package spacemesh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOpenConcurrent(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	finished := 0

	for i := 0; i < 2; i++ {
		go func() {
			p.getConnection(234)
			finished++
		}()
	}

	time.Sleep(openConnectionDelay + 100*time.Millisecond)
	assert.Equal(t, 2, finished)
}


func TestUseCachedConn(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	finished := false

	go func() {
		p.getConnection(234)
		p.getConnection(234)
		finished = true
	}()

	time.Sleep(openConnectionDelay + 100*time.Millisecond)
	assert.True(t, finished)
}
