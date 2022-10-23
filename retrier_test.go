package graphql

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCountRetrier(t *testing.T) {
	t.Parallel()
	retrier := NewCountRetrier(1)
	someErr := errors.New("some error")

	assert.Equal(t, CountRetrier{
		maxAttempts: 2,
	}, retrier)
	assert.True(t, retrier.ShouldRetry(someErr, 1))
	assert.False(t, retrier.ShouldRetry(someErr, 2))
}
