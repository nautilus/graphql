package graphql

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

func formatIndentPrefix(level int) string {
	acc := "\n"
	// build up the prefix
	for i := 0; i <= level; i++ {
		acc += "    "
	}

	return acc
}
func formatSelectionSelectionSet(level int, selectionSet ast.SelectionSet) string {
	acc := " {"
	// and any sub selection
	acc += formatSelection(level+1, selectionSet)
	acc += formatIndentPrefix(level) + "}"

	return acc
}

func formatSelection(level int, selectionSet ast.SelectionSet) string {
	acc := ""

	for _, selection := range selectionSet {
		acc += formatIndentPrefix(level)
		switch selection := selection.(type) {
		case *ast.Field:
			// add the field name
			acc += selection.Name
			if len(selection.SelectionSet) > 0 {
				acc += formatSelectionSelectionSet(level, selection.SelectionSet)
			}
		case *ast.InlineFragment:
			// print the fragment name
			acc += fmt.Sprintf("... on %v", selection.TypeCondition) +
				formatSelectionSelectionSet(level, selection.SelectionSet)
		case *ast.FragmentSpread:
			// print the fragment name
			acc += "..." + selection.Name
		}
	}

	return acc
}

// FormatSelectionSet returns a pretty printed version of a selection set
func FormatSelectionSet(selection ast.SelectionSet) string {
	acc := "{"

	insides := formatSelection(0, selection)

	if strings.TrimSpace(insides) != "" {
		acc += insides + "\n}"
	} else {
		acc += "}"
	}

	return acc
}
