package spacemesh

import "sync"

func (p *connectionPool) shutdown() {
	wg := sync.WaitGroup{}

	// stop routines currently opening a connection and prevent new connections
	close(p.shuttingDown)

	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	wg.Add(len(p.cache))
	for _, conn := range p.cache {
		conn := conn
		go func() {
			conn.Close()
			wg.Done()
		}()
	}
	wg.Wait()

	p.cache = nil
}
