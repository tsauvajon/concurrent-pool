package spacemesh


// Try to store an open connection, otherwise used the already cached one
func (p *connectionPool) storeToCache(ipAddress int32, conn Connection) Connection {
	p.cacheMx.Lock()
	defer p.cacheMx.Unlock()

	if cachedConn, ok := p.getFromCache(ipAddress); ok {
		return cachedConn
	}

	p.cacheMx.Lock()
	p.cache[ipAddress] = conn
	p.cacheMx.Unlock()

	return conn
}

