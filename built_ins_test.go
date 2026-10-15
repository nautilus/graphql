package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBuiltInDirective(t *testing.T) {
	t.Parallel()
	assert.True(t, isBuiltInDirective("specifiedBy"))
	assert.True(t, isBuiltInDirective("defer"))
	assert.False(t, isBuiltInDirective("foo"))
}

func TestIsBuiltInType(t *testing.T) {
	t.Parallel()
	assert.True(t, isBuiltInType("Int"))
	assert.True(t, isBuiltInType("__Schema"))
	assert.False(t, isBuiltInType("Foo"))
}
