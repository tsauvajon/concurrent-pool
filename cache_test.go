package spacemesh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Parallel()

	p := newConnectionPool()
	p.getConnection(234)

	c, ok := p.cache[234]
	assert.True(t, ok)
	assert.NotNil(t, c)
}
