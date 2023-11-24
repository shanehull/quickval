package quickfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewQuickFS_Test(t *testing.T) {
	q := NewQuickFS(WithAPIKey("key"), WithFCF(), WithFYHistory(5))

	assert.Equal(t, "key", q.apiKey)
	assert.True(t, q.fcf)
	assert.Equal(t, 5, q.fyHistory)
}
