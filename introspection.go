package graphql

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2/ast"
)

// IntrospectOptions represents the options for the IntrospectAPI function
type IntrospectOptions struct {
	// mergeFunc is an option-specific merger. This makes adding new options easier.
	// If non-nil (i.e. created by an introspection func here), then sets its own options into opts.
	mergeFunc func(opts *IntrospectOptions)

	client  *http.Client
	ctx     context.Context
	retrier Retrier
	wares   []NetworkMiddleware
}

// Context returns either a given context or an instance of the context.Background
func (o *IntrospectOptions) Context() context.Context {
	if o.ctx == nil {
		return context.Background()
	}
	return o.ctx
}

// Apply applies the options to a given queryer
func (o *IntrospectOptions) Apply(queryer Queryer) Queryer {
	if q, ok := queryer.(QueryerWithMiddlewares); ok && len(o.wares) > 0 {
		queryer = q.WithMiddlewares(o.wares)
	}
	if q, ok := queryer.(HTTPQueryer); ok && o.client != nil {
		queryer = q.WithHTTPClient(o.client)
	}
	return queryer
}

func mergeIntrospectOptions(opts ...*IntrospectOptions) *IntrospectOptions {
	res := &IntrospectOptions{}
	for _, opt := range opts {
		if opt.mergeFunc != nil { // Verify non-nil. Previously did not require mergeFuncs. so could panic if client code uses raw "&IntrospectOptions{}".
			opt.mergeFunc(res)
		}
	}
	return res
}

// IntrospectWithMiddlewares returns an instance of graphql.IntrospectOptions with given middlewares
// to be pass to an instance of a graphql.Queryer by the IntrospectOptions.Apply function
func IntrospectWithMiddlewares(wares ...NetworkMiddleware) *IntrospectOptions {
	return introspectOptsFunc(func(opts *IntrospectOptions) {
		opts.wares = append(opts.wares, wares...)
	})
}

// IntrospectWithHTTPClient returns an instance of graphql.IntrospectOptions with given client
// to be pass to an instance of a graphql.Queryer by the IntrospectOptions.Apply function
func IntrospectWithHTTPClient(client *http.Client) *IntrospectOptions {
	return introspectOptsFunc(func(opts *IntrospectOptions) {
		opts.client = client
	})
}

func introspectOptsFunc(fn func(opts *IntrospectOptions)) *IntrospectOptions {
	opts := &IntrospectOptions{mergeFunc: fn}
	opts.mergeFunc(opts)
	return opts
}

// IntrospectWithHTTPClient returns an instance of graphql.IntrospectOptions with given context
// to be used as a parameter for graphql.Queryer.Query function in the graphql.IntrospectAPI function
func IntrospectWithContext(ctx context.Context) *IntrospectOptions {
	return introspectOptsFunc(func(opts *IntrospectOptions) {
		opts.ctx = ctx
	})
}

// IntrospectWithRetrier returns an instance of graphql.IntrospectOptions with the given Retrier.
// For a fixed number of retries, see CountRetrier.
func IntrospectWithRetrier(retrier Retrier) *IntrospectOptions {
	return introspectOptsFunc(func(opts *IntrospectOptions) {
		opts.retrier = retrier
	})
}

// IntrospectRemoteSchema is used to build a RemoteSchema by firing the introspection query
// at a remote service and reconstructing the schema object from the response
func IntrospectRemoteSchema(url string, opts ...*IntrospectOptions) (*RemoteSchema, error) {

	// introspect the schema at the designated url
	schema, err := IntrospectAPI(NewSingleRequestQueryer(url), opts...)
	if err != nil {
		return nil, err
	}

	return &RemoteSchema{
		URL:    url,
		Schema: schema,
	}, nil
}

// IntrospectRemoteSchemas takes a list of URLs and creates a RemoteSchema by invoking
// graphql.IntrospectRemoteSchema at that location.
func IntrospectRemoteSchemas(urls ...string) ([]*RemoteSchema, error) {
	return IntrospectRemoteSchemasWithOptions(urls)
}

// IntrospectRemoteSchemasWithOptions takes a list of URLs and an optional list of graphql.IntrospectionOptions
// and creates a RemoteSchema by invoking graphql.IntrospectRemoteSchema at that location.
func IntrospectRemoteSchemasWithOptions(urls []string, opts ...*IntrospectOptions) ([]*RemoteSchema, error) {
	// build up the list of remote schemas
	schemas := []*RemoteSchema{}

	for _, service := range urls {
		// introspect the locations
		schema, err := IntrospectRemoteSchema(service, opts...)
		if err != nil {
			return nil, err
		}

		// add the schema to the list
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// IntrospectAPI send the introspection query to a Queryer and builds up the
// schema object described by the result
func IntrospectAPI(queryer Queryer, opts ...*IntrospectOptions) (*ast.Schema, error) {
	// apply the options to the given queryer
	opt := mergeIntrospectOptions(opts...)
	queryer = opt.Apply(queryer)

	query := func() (IntrospectionQueryResult, error) {
		var result IntrospectionQueryResult
		input := &QueryInput{
			Query:         IntrospectionQuery,
			OperationName: "IntrospectionQuery",
		}
		err := queryer.Query(opt.Context(), input, &result)
		return result, errors.WithMessage(err, "query failed")
	}
	// fire the introspection query
	result, err := query()
	if opt.retrier != nil {
		// if available, retry on failures
		var attempts uint = 1
		for err != nil && opt.retrier.ShouldRetry(err, attempts) {
			result, err = query()
			attempts++
		}
	}
	if err != nil {
		return nil, err
	}

	// grab the schema
	remoteSchema := result.Schema

	// create a schema we will build up over time
	schema := &ast.Schema{
		Types:         map[string]*ast.Definition{},
		Directives:    map[string]*ast.DirectiveDefinition{},
		PossibleTypes: map[string][]*ast.Definition{},
		Implements:    map[string][]*ast.Definition{},
	}

	// if we dont have a name on the response
	if remoteSchema == nil || remoteSchema.QueryType.Name == "" {
		return nil, errors.New("Could not find the root query")
	}

	// reconstructing the schema happens in a few pass throughs
	// the first builds a map of type names to their definition
	// the second pass goes over the definitions and reconstructs the types

	// add each type to the schema
	for _, remoteType := range remoteSchema.Types {
		// convert turn the API payload into a schema type
		schemaType := introspectionUnmarshalType(remoteType)

		// check if this type is the QueryType
		if remoteType.Name == remoteSchema.QueryType.Name {
			schema.Query = schemaType
		} else if remoteSchema.MutationType != nil && schemaType.Name == remoteSchema.MutationType.Name {
			schema.Mutation = schemaType
		} else if remoteSchema.SubscriptionType != nil && schemaType.Name == remoteSchema.SubscriptionType.Name {
			schema.Subscription = schemaType
		}

		// register the type with the schema
		schema.Types[schemaType.Name] = schemaType
	}

	// the second pass constructs the fields and
	for _, remoteType := range remoteSchema.Types {
		// a reference to the type
		storedType, ok := schema.Types[remoteType.Name]
		if !ok {
			return nil, err
		}

		// make sure we record that a type implements itself
		schema.AddImplements(remoteType.Name, storedType)
		if storedType.Kind == ast.Object {
			addPossibleTypeOnce(schema, remoteType.Name, storedType) // When evaluating matching fragments, Objects count as a possible type for themselves.
		}

		// if we are looking at an enum
		if len(remoteType.PossibleTypes) > 0 {
			// build up an empty list of union types
			storedType.Types = []string{}

			// each union value needs to be added to the list
			for _, possibleType := range remoteType.PossibleTypes {
				// if there is no name
				if possibleType.Name == "" {
					return nil, errors.New("Could not find name of type")
				}

				possibleTypeDef, ok := schema.Types[possibleType.Name]
				if !ok {
					return nil, errors.New("Could not find type definition for union implementation")
				}

				// skip the type, if the name equals the current one
				if possibleType.Name == storedType.Name {
					continue
				}
				if storedType.Kind == ast.Union {
					storedType.Types = append(storedType.Types, possibleType.Name)
				}

				// add the possible type to the schema
				addPossibleTypeOnce(schema, remoteType.Name, possibleTypeDef)
				schema.AddImplements(possibleType.Name, storedType)
			}
		}

		if len(remoteType.Interfaces) > 0 {

			// each interface value needs to be added to the list
			for _, iFace := range remoteType.Interfaces {
				// if there is no name
				if iFace.Name == "" {
					return nil, errors.New("Could not find name of type")
				}

				// add the type to the union definition
				storedType.Interfaces = append(storedType.Interfaces, iFace.Name)

				iFaceDef, ok := schema.Types[iFace.Name]
				if !ok {
					return nil, errors.New("Could not find type definition for union implementation")
				}

				// add the possible type to the schema
				addPossibleTypeOnce(schema, iFaceDef.Name, storedType)
				schema.AddImplements(storedType.Name, iFaceDef)
			}
		}

		// build up a list of fields associated with the type
		fields := ast.FieldList{}

		for _, field := range remoteType.Fields {
			// add the field to the list
			fields = append(fields, &ast.FieldDefinition{
				Name:        field.Name,
				Type:        introspectionUnmarshalTypeRef(&field.Type),
				Description: field.Description,
				Arguments:   introspectionConvertArgList(field.Args),
			})
		}

		for _, field := range remoteType.InputFields {
			// add the field to the list
			fields = append(fields, &ast.FieldDefinition{
				Name:        field.Name,
				Type:        introspectionUnmarshalTypeRef(&field.Type),
				Description: field.Description,
			})
		}

		// save the list of fields in the schema type
		storedType.Fields = fields
	}

	// add each directive to the schema
	for _, directive := range remoteSchema.Directives {
		// if we dont have a name
		if directive.Name == "" {
			return nil, errors.New("could not find directive name")
		}

		// the list of directive locations
		locations, err := introspectionUnmarshalDirectiveLocation(directive.Locations)
		if err != nil {
			return nil, err
		}

		// save the directive definition to the schema
		schema.Directives[directive.Name] = &ast.DirectiveDefinition{
			Position:    &ast.Position{Src: &ast.Source{}},
			Name:        directive.Name,
			Description: directive.Description,
			Arguments:   introspectionConvertArgList(directive.Args),
			Locations:   locations,
		}
		switch directive.Name {
		case "skip", "deprecated", "include":
			schema.Directives[directive.Name].Position.Src.BuiltIn = true
		}
	}

	// we're done here
	return schema, nil
}

func addPossibleTypeOnce(schema *ast.Schema, name string, definition *ast.Definition) {
	for _, typ := range schema.PossibleTypes[name] {
		if typ.Name == definition.Name {
			return
		}
	}
	schema.AddPossibleType(name, definition)
}

func introspectionConvertArgList(args []IntrospectionInputValue) ast.ArgumentDefinitionList {
	result := ast.ArgumentDefinitionList{}

	// we need to add each argument to the field
	for _, argument := range args {
		result = append(result, &ast.ArgumentDefinition{
			Name:        argument.Name,
			Description: argument.Description,
			Type:        introspectionUnmarshalTypeRef(&argument.Type),
		})
	}

	return result
}

func introspectionUnmarshalType(schemaType IntrospectionQueryFullType) *ast.Definition {
	definition := &ast.Definition{
		Name:        schemaType.Name,
		Description: schemaType.Description,
	}

	// the kind of type
	switch schemaType.Kind {
	case "OBJECT":
		definition.Kind = ast.Object
	case "SCALAR":
		definition.Kind = ast.Scalar
	case "INTERFACE":
		definition.Kind = ast.Interface
	case "UNION":
		definition.Kind = ast.Union
	case "INPUT_OBJECT":
		definition.Kind = ast.InputObject
	case "ENUM":
		definition.Kind = ast.Enum
		// save the enum values
		definition.EnumValues = ast.EnumValueList{}

		// convert each enum value into the appropriate object
		for _, value := range schemaType.EnumValues {
			definition.EnumValues = append(definition.EnumValues, &ast.EnumValueDefinition{
				Name:        value.Name,
				Description: value.Description,
			})
		}
	}
	switch schemaType.Name {
	case "ID", "Int", "Float", "String", "Boolean",
		"__Schema", "__Type", "__InputValue", "__TypeKind",
		"__DirectiveLocation", "__Field", "__EnumValue", "__Directive":
		definition.BuiltIn = true
	}
	return definition
}

// a mapping of marshaled directive locations to their parsed equivalent
var directiveLocationMap map[string]ast.DirectiveLocation

func introspectionUnmarshalDirectiveLocation(locs []string) ([]ast.DirectiveLocation, error) {
	result := []ast.DirectiveLocation{}

	// each location needs to be mapped over
	for _, value := range locs {
		// look up the directive location for the API response
		location, ok := directiveLocationMap[value]
		if !ok {
			return nil, fmt.Errorf("encountered unknown directive location: %s", value)
		}

		// add the result to the list
		result = append(result, location)
	}

	// we're done
	return result, nil
}

func introspectionUnmarshalTypeRef(response *IntrospectionTypeRef) *ast.Type {
	// we could have a non-null list of a field
	if response.Kind == "NON_NULL" && response.OfType.Kind == "LIST" {
		return ast.NonNullListType(introspectionUnmarshalTypeRef(response.OfType.OfType), &ast.Position{})
	}

	// we could have a list of a type
	if response.Kind == "LIST" {
		return ast.ListType(introspectionUnmarshalTypeRef(response.OfType), &ast.Position{})
	}

	// we could have just a non null
	if response.Kind == "NON_NULL" {
		return ast.NonNullNamedType(response.OfType.Name, &ast.Position{})
	}

	// if we are looking at a named type that isn't in a list or marked non-null
	return ast.NamedType(response.Name, &ast.Position{})
}

func init() {
	directiveLocationMap = map[string]ast.DirectiveLocation{
		"QUERY":                  ast.LocationQuery,
		"MUTATION":               ast.LocationMutation,
		"SUBSCRIPTION":           ast.LocationSubscription,
		"FIELD":                  ast.LocationField,
		"FRAGMENT_DEFINITION":    ast.LocationFragmentDefinition,
		"FRAGMENT_SPREAD":        ast.LocationFragmentSpread,
		"INLINE_FRAGMENT":        ast.LocationInlineFragment,
		"SCHEMA":                 ast.LocationSchema,
		"SCALAR":                 ast.LocationScalar,
		"OBJECT":                 ast.LocationObject,
		"FIELD_DEFINITION":       ast.LocationFieldDefinition,
		"ARGUMENT_DEFINITION":    ast.LocationArgumentDefinition,
		"INTERFACE":              ast.LocationInterface,
		"UNION":                  ast.LocationUnion,
		"ENUM":                   ast.LocationEnum,
		"ENUM_VALUE":             ast.LocationEnumValue,
		"INPUT_OBJECT":           ast.LocationInputObject,
		"INPUT_FIELD_DEFINITION": ast.LocationInputFieldDefinition,
	}
}

type IntrospectionQueryResult struct {
	Schema *IntrospectionQuerySchema `json:"__schema"`
}

type IntrospectionQuerySchema struct {
	QueryType        IntrospectionQueryRootType    `json:"queryType"`
	MutationType     *IntrospectionQueryRootType   `json:"mutationType"`
	SubscriptionType *IntrospectionQueryRootType   `json:"subscriptionType"`
	Types            []IntrospectionQueryFullType  `json:"types"`
	Directives       []IntrospectionQueryDirective `json:"directives"`
}

type IntrospectionQueryDirective struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Locations   []string                  `json:"locations"`
	Args        []IntrospectionInputValue `json:"args"`
}

type IntrospectionQueryRootType struct {
	Name string `json:"name"`
}

type IntrospectionQueryFullTypeField struct {
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	Args              []IntrospectionInputValue `json:"args"`
	Type              IntrospectionTypeRef      `json:"type"`
	IsDeprecated      bool                      `json:"isDeprecated"`
	DeprecationReason string                    `json:"deprecationReason"`
}

type IntrospectionQueryFullType struct {
	Kind          string                             `json:"kind"`
	Name          string                             `json:"name"`
	Description   string                             `json:"description"`
	InputFields   []IntrospectionInputValue          `json:"inputFields"`
	Interfaces    []IntrospectionTypeRef             `json:"interfaces"`
	PossibleTypes []IntrospectionTypeRef             `json:"possibleTypes"`
	Fields        []IntrospectionQueryFullTypeField  `json:"fields"`
	EnumValues    []IntrospectionQueryEnumDefinition `json:"enumValues"`
}

type IntrospectionQueryEnumDefinition struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

type IntrospectionInputValue struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	DefaultValue string               `json:"defaultValue"`
	Type         IntrospectionTypeRef `json:"type"`
}

type IntrospectionTypeRef struct {
	Kind   string                `json:"kind"`
	Name   string                `json:"name"`
	OfType *IntrospectionTypeRef `json:"ofType"`
}

// IntrospectionQuery is the query that is fired at an API to reconstruct its schema
var IntrospectionQuery = `
	query IntrospectionQuery {
		__schema {
			queryType { name }
			mutationType { name }
			subscriptionType { name }
			types {
				...FullType
			}
			directives {
				name
				description
				locations
				args {
				...InputValue
				}
			}
		}
	}

	fragment FullType on __Type {
		kind
		name
		description
		fields(includeDeprecated: true) {
			name
			description
			args {
				...InputValue
			}
			type {
				...TypeRef
			}
			isDeprecated
			deprecationReason
		}

		inputFields {
			...InputValue
		}

		interfaces {
			...TypeRef
		}

		enumValues(includeDeprecated: true) {
			name
			description
			isDeprecated
			deprecationReason
		}
		possibleTypes {
			...TypeRef
		}
	}

	fragment InputValue on __InputValue {
		name
		description
		type { ...TypeRef }
		defaultValue
	}

	fragment TypeRef on __Type {
		kind
		name
		ofType {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
							ofType {
								kind
								name
								ofType {
									kind
									name
								}
							}
						}
					}
				}
			}
		}
	}
`
