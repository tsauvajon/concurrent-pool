package spacemesh

// Get an opened connection from cache, if it exists
func (p *connectionPool) readFromCache(ipAddress int32) (Connection, bool) {
	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	conn, ok := p.cache[ipAddress]
	return conn, ok
}

// Try to store an open connection, otherwise use the already cached one.
// Returns a connection, and a boolean: true if we keep the connection
// passed as parameter, false if we keep the cached connection.
func (p *connectionPool) writeToCache(ipAddress int32, conn Connection) (Connection, bool) {
	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	// return the cached conn, don't overwrite
	if cachedConn, ok := p.cache[ipAddress]; ok {
		return cachedConn, false
	}

	p.cache[ipAddress] = conn
	return conn, true
}
