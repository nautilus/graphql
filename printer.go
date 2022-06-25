package graphql

import (
	"bytes"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

// PrintQuery creates a string representation of an operation
func PrintQuery(document *ast.QueryDocument) (string, error) {
	var buf bytes.Buffer
	formatter.NewFormatter(&buf).FormatQueryDocument(document)
	return buf.String(), nil
}
