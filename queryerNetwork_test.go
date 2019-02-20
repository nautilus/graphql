package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSingleRequestQueryer(t *testing.T) {
	// make sure that create a new query renderer saves the right URL
	assert.Equal(t, "foo", NewSingleRequestQueryer("foo").queryer.URL)
}
