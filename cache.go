package spacemesh

// Get an opened connection from cache, if it exists
func (p *connectionPool) getFromCache(ipAddress int32) (Connection, bool) {
	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	conn, ok := p.cache[ipAddress]
	return conn, ok
}

// Try to store an open connection, otherwise used the already cached one
func (p *connectionPool) storeToCache(ipAddress int32, conn Connection) Connection {
	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	// return the cached conn, don't overwrite
	if cachedConn, ok := p.cache[ipAddress]; ok {
		return cachedConn
	}

	p.cache[ipAddress] = conn
	return conn
}
