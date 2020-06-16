package spacemesh

import (
	"context"
	"sync"
	"time"
)

type Connection interface {
	Open(ctx context.Context)
	Close()
}

type connection struct {
	openingDelay time.Duration

	isOpenMx sync.RWMutex
	isOpen   bool
}

func (c *connection) Open(ctx context.Context) {
	select {
	case <-time.After(c.openingDelay):
		// the connection opened successfully
		c.isOpenMx.Lock()
		defer c.isOpenMx.Unlock()
		c.isOpen = true
	case <-ctx.Done():
		// connection opening was aborted
		c.Close()
	}
}

func (c *connection) Close() {
	c.isOpenMx.Lock()
	defer c.isOpenMx.Unlock()

	if !c.isOpen {
		return
	}

	c.isOpen = false
}
