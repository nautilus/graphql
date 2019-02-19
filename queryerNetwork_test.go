package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNetworkQueryer(t *testing.T) {
	// make sure that create a new query renderer saves the right URL
	assert.Equal(t, "foo", NewNetworkQueryer("foo").URL)
}
