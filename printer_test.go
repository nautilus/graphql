package graphql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestPrintQuery(t *testing.T) {
	table := []struct {
		name     string
		expected string
		query    *ast.QueryDocument
	}{
		{
			name: "single root field",
			expected: `query {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "hello",
							},
						},
					},
				},
			},
		},
		{
			name: "variable values",
			expected: `query {
	hello(foo: $foo)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "hello",
								Arguments: ast.ArgumentList{
									&ast.Argument{
										Name: "foo",
										Value: &ast.Value{
											Kind: ast.Variable,
											Raw:  "foo",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "directives",
			expected: `query {
	hello @foo(bar: "baz")
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Directives: ast.DirectiveList{
								&ast.Directive{
									Name: "foo",
									Arguments: ast.ArgumentList{
										&ast.Argument{
											Name: "bar",
											Value: &ast.Value{
												Kind: ast.StringValue,
												Raw:  "baz",
											},
										},
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "directives",
			expected: `query {
	... on User @foo {
		hello
	}
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.InlineFragment{
								TypeCondition: "User",
								SelectionSet: ast.SelectionSet{
									&ast.Field{
										Name: "hello",
									},
								},
								Directives: ast.DirectiveList{
									&ast.Directive{
										Name: "foo",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple root fields",
			expected: `query {
	hello
	goodbye
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "hello",
							},
							&ast.Field{
								Name: "goodbye",
							},
						},
					},
				},
			},
		},
		{
			name: "selection set",
			expected: `query {
	hello {
		world
	}
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							SelectionSet: ast.SelectionSet{
								&ast.Field{
									Name: "world",
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "inline fragments",
			expected: `query {
	... on Foo {
		hello
	}
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.InlineFragment{
								TypeCondition: "Foo",
								SelectionSet: ast.SelectionSet{
									&ast.Field{
										Name: "hello",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "fragments",
			expected: `query {
	... Foo
}
fragment Foo on User {
	firstName
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Query,
						SelectionSet: ast.SelectionSet{
							&ast.FragmentSpread{
								Name: "Foo",
							},
						},
					},
				},
				Fragments: ast.FragmentDefinitionList{
					&ast.FragmentDefinition{
						Name: "Foo",
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "firstName",
								Definition: &ast.FieldDefinition{
									Type: ast.NamedType("String", &ast.Position{}),
								},
							},
						},
						TypeCondition: "User",
					},
				},
			},
		},
		{
			name: "alias",
			expected: `query {
	bar: hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name:  "hello",
							Alias: "bar",
						},
					},
				},
				},
			},
		},
		{
			name: "string arguments",
			expected: `query {
	hello(hello: "world")
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.StringValue,
										Raw:  "world",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "json string arguments",
			expected: `query {
	hello(json: "{\"foo\": \"bar\"}")
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "json",
									Value: &ast.Value{
										Kind: ast.StringValue,
										Raw:  "{\"foo\": \"bar\"}",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "int arguments",
			expected: `query {
	hello(hello: 1)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.IntValue,
										Raw:  "1",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "boolean arguments",
			expected: `query {
	hello(hello: true)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.BooleanValue,
										Raw:  "true",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "variable arguments",
			expected: `query {
	hello(hello: $hello)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.IntValue,
										Raw:  "$hello",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "null arguments",
			expected: `query {
	hello(hello: null)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Raw:  "null",
										Kind: ast.NullValue,
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "float arguments",
			expected: `query {
	hello(hello: 1.1)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.FloatValue,
										Raw:  "1.1",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "enum arguments",
			expected: `query {
	hello(hello: Hello)
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.EnumValue,
										Raw:  "Hello",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "list arguments",
			expected: `query {
	hello(hello: ["hello",1])
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.ListValue,
										Children: ast.ChildValueList{
											{
												Value: &ast.Value{
													Kind: ast.StringValue,
													Raw:  "hello",
												},
											},
											{
												Value: &ast.Value{
													Kind: ast.IntValue,
													Raw:  "1",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "object arguments",
			expected: `query {
	hello(hello: {hello:"hello",goodbye:1})
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.ObjectValue,
										Children: ast.ChildValueList{
											{
												Name: "hello",
												Value: &ast.Value{
													Kind: ast.StringValue,
													Raw:  "hello",
												},
											},
											{
												Name: "goodbye",
												Value: &ast.Value{
													Kind: ast.IntValue,
													Raw:  "1",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "multiple arguments",
			expected: `query {
	hello(hello: "world", goodbye: "moon")
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
							Arguments: ast.ArgumentList{
								{
									Name: "hello",
									Value: &ast.Value{
										Kind: ast.StringValue,
										Raw:  "world",
									},
								},
								{
									Name: "goodbye",
									Value: &ast.Value{
										Kind: ast.StringValue,
										Raw:  "moon",
									},
								},
							},
						},
					},
				},
				},
			},
		},
		{
			name: "anonymous variables to query",
			expected: `query ($id: ID!) {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
						},
					},
					VariableDefinitions: ast.VariableDefinitionList{
						&ast.VariableDefinition{
							Variable: "id",
							Type: &ast.Type{
								NamedType: "ID",
								NonNull:   true,
							},
						},
					},
				},
				},
			},
		},
		{
			name: "named query with variables",
			expected: `query foo ($id: String!) {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					Name:      "foo",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
						},
					},
					VariableDefinitions: ast.VariableDefinitionList{
						&ast.VariableDefinition{
							Variable: "id",
							Type: &ast.Type{
								NamedType: "String",
								NonNull:   true,
							},
						},
					},
				},
				},
			},
		},
		{
			name: "named query with variables",
			expected: `query foo ($id: [String]) {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					Name:      "foo",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
						},
					},
					VariableDefinitions: ast.VariableDefinitionList{
						&ast.VariableDefinition{
							Variable: "id",
							Type:     ast.ListType(ast.NamedType("String", &ast.Position{}), &ast.Position{}),
						},
					},
				},
				},
			},
		},
		{
			name: "named query with variables",
			expected: `query foo ($id: [String!]) {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Query,
					Name:      "foo",
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
						},
					},
					VariableDefinitions: ast.VariableDefinitionList{
						&ast.VariableDefinition{
							Variable: "id",
							Type:     ast.ListType(ast.NonNullNamedType("String", &ast.Position{}), &ast.Position{}),
						},
					},
				},
				},
			},
		},
		{
			name: "single mutation field",
			expected: `mutation {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{&ast.OperationDefinition{
					Operation: ast.Mutation,
					SelectionSet: ast.SelectionSet{
						&ast.Field{
							Name: "hello",
						},
					},
				},
				},
			},
		},
		{
			name: "single subscription field",
			expected: `subscription {
	hello
}
`,
			query: &ast.QueryDocument{
				Operations: ast.OperationList{
					&ast.OperationDefinition{
						Operation: ast.Subscription,
						SelectionSet: ast.SelectionSet{
							&ast.Field{
								Name: "hello",
							},
						},
					},
				},
			},
		},
	}

	for _, row := range table {
		t.Run(row.name, func(t *testing.T) {
			str, err := PrintQuery(row.query)
			if err != nil {
				t.Error(err.Error())
			}

			assert.Equal(t, row.expected, str)
		})
	}
}
