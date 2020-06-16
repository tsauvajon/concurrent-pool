package spacemesh

import (
	"context"
	"time"
)

type Connection interface {
	Open(ctx context.Context)
	Close()
}

type connection struct {
	openingDelay time.Duration
	isOpen       bool
}

func (c *connection) Open(ctx context.Context)  {
	select {
		case <-time.After(c.openingDelay):
			c.isOpen = true
		case <- ctx.Done():
			// real world: handle errors/closing
	}
}
func (c *connection) Close() {
	c.isOpen = false
}
