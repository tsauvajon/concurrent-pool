package spacemesh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOpenClose(t *testing.T) {
	t.Parallel()

	c := &connection{}
	c.Open(context.Background())
	assert.True(t, c.isOpen)

	c.Close()
	assert.False(t, c.isOpen)

	c.Close()
	assert.False(t, c.isOpen)
}

func TestCancelOpen(t *testing.T) {
	t.Parallel()

	conn := &connection{
		openingDelay: 1 * time.Second,
	}
	done := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		conn.Open(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		assert.False(t, conn.isOpen)
	case <-time.After(2 * time.Second):
		t.Fail()
	}
}
