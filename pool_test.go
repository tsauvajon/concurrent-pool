package spacemesh

//
import (
	"testing"
	"time"
)

func TestOpenConcurrent(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	finished := make(chan struct{}, 2)

	for i := 0; i < 2; i++ {
		go func() {
			p.getConnection(234)
			finished <- struct{}{}
		}()
	}

	done := make(chan struct{}, 1)

	go func() {
		for i := 0; i < 2; i++ {
			<-finished
			done <- struct{}{}
		}
	}()

	select {
	case <-done:
		return
	case <-time.After(openConnectionDelay + 100*time.Millisecond):
		t.Fail()
	}
}

func TestUseCachedConn(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	done := make(chan struct{}, 1)

	go func() {
		p.getConnection(234)
		p.getConnection(234)
		done <- struct{}{}
	}()

	select {
	case <-done:
		return
	case <-time.After(openConnectionDelay + 100*time.Millisecond):
		t.Fail()
	}
}
