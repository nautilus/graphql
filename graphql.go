package graphql

import (
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// LoadSchema takes an SDL string and returns the parsed version
func LoadSchema(typedef string) (*ast.Schema, error) {
	schema, err := gqlparser.LoadSchema(&ast.Source{
		Input: typedef,
	})

	// vektah/gqlparser returns non-nil err all the time
	if schema == nil {
		return nil, err
	}
	return schema, nil
}
