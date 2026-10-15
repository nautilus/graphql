package graphql

import (
	"sync"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/validator"
)

var preludeSchema = sync.OnceValues(func() (*ast.Schema, error) {
	return validator.LoadSchema(validator.Prelude)
})

func mustBuiltInSchema() *ast.Schema {
	prelude, err := preludeSchema()
	if err != nil {
		panic(err)
	}
	return prelude
}

func isBuiltInDirective(name string) bool {
	directive, ok := mustBuiltInSchema().Directives[name]
	return ok && directive.Position.Src.BuiltIn
}

func isBuiltInType(name string) bool {
	typ, ok := mustBuiltInSchema().Types[name]
	return ok && typ.Position.Src.BuiltIn
}
