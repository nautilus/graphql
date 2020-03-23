package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFormatSelectionSet(t *testing.T) {
	// the table of sets to test
	rows := []struct {
		input    ast.SelectionSet
		expected string
	}{
		{
			ast.SelectionSet{&ast.Field{Name: "firstName"}},
			`{
    firstName
}`,
		},
	}

	for _, row := range rows {
		// make sure we get the expected result
		assert.Equal(t, row.expected, FormatSelectionSet(row.input))
	}
}
