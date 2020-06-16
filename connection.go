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
	isOpen       bool

	isOpenMx sync.RWMutex
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
		// real world: handle errors/closing
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
