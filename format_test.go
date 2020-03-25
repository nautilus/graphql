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
			ast.SelectionSet{},
			"{}",
		},
		{
			ast.SelectionSet{
				&ast.Field{Name: "firstName"},
				&ast.Field{Name: "friend", SelectionSet: ast.SelectionSet{&ast.Field{Name: "lastName"}}},
			},
			`{
    firstName
    friend {
        lastName
    }
}`,
		},
		{
			ast.SelectionSet{&ast.FragmentSpread{Name: "MyFragment"}},
			`{
    ...MyFragment
}`,
		},
		{
			ast.SelectionSet{
				&ast.InlineFragment{
					TypeCondition: "MyType",
					SelectionSet:  ast.SelectionSet{&ast.Field{Name: "firstName"}},
				},
			},
			`{
    ... on MyType {
        firstName
    }
}`,
		},
	}

	for _, row := range rows {
		// make sure we get the expected result
		assert.Equal(t, row.expected, FormatSelectionSet(row.input))
	}
}
