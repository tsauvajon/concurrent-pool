package spacemesh

import "time"

type Connection interface {
	Open()
	Close()
}

type connection struct {

}

var openConnectionDelay = 1*time.Second

func (c *connection) Open()  {
	time.Sleep(openConnectionDelay)
}
func (c *connection) Close() {}
