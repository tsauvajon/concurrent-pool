package spacemesh

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenClose(t *testing.T) {
	t.Parallel()

	c := &connection{}
	c.Open(context.Background())
	assert.True(t, c.isOpen)

	c.Close()
	assert.False(t, c.isOpen)
}

func TestCancelOpen(t *testing.T) {
	t.Parallel()

	c := &connection{}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.Open(ctx)
	}()

	cancel()
	assert.False(t, c.isOpen)
}
