package graphql

import (
	"context"
	"reflect"

	"github.com/vektah/gqlparser/ast"
)

// RemoteSchema encapsulates a particular schema that can be executed by sending network requests to the
// specified URL.
type RemoteSchema struct {
	Schema *ast.Schema
	URL    string
}

// QueryInput provides all of the information required to fire a query
type QueryInput struct {
	Query         string
	QueryDocument *ast.QueryDocument
	OperationName string
	Variables     map[string]interface{}
}

// Queryer is a interface for objects that can perform
type Queryer interface {
	Query(context.Context, *QueryInput, interface{}) error
}

// QueryerWithMiddlewares is an interface for queryers that support network middlewares
type QueryerWithMiddlewares interface {
	WithMiddlewares(mwares []NetworkMiddleware) Queryer
}

// Provided Implementations

// MockSuccessQueryer responds with pre-defined value when executing a query
type MockSuccessQueryer struct {
	Value interface{}
}

// Query looks up the name of the query in the map of responses and returns the value
func (q *MockSuccessQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	// assume the mock is writing the same kind as the receiver
	reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(q.Value))

	// this will panic if something goes wrong
	return nil
}

// QueryerFunc responds to the query by calling the provided function
type QueryerFunc func(*QueryInput) (interface{}, error)

// Query invokes the provided function and writes the response to the receiver
func (q QueryerFunc) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	// invoke the handler
	response, err := q(input)
	if err != nil {
		return err
	}

	// assume the mock is writing the same kind as the receiver
	reflect.ValueOf(receiver).Elem().Set(reflect.ValueOf(response))

	// no errors
	return nil
}

// IntrospectRemoteSchema is used to build a RemoteSchema by firing the introspection query
// at a remote service and reconstructing the schema object from the response
func IntrospectRemoteSchema(url string) (*RemoteSchema, error) {
	// introspect the schema at the designated url
	schema, err := IntrospectAPI(NewNetworkQueryer(url))
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
	// build up the list of remote schemas
	schemas := []*RemoteSchema{}

	for _, service := range urls {
		// introspect the locations
		schema, err := IntrospectRemoteSchema(service)
		if err != nil {
			return nil, err
		}

		// add the schema to the list
		schemas = append(schemas, schema)
	}

	return schemas, nil
}
