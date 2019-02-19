package graphql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMultiOpQueryer(t *testing.T) {
	queryer := NewMultiOpQueryer("foo", 1*time.Millisecond, 100)

	// make sure the queryer config is all correct
	assert.Equal(t, "foo", queryer.URL)
	assert.Equal(t, 1*time.Millisecond, queryer.BatchInterval)
	assert.Equal(t, 100, queryer.MaxBatchSize)
}
