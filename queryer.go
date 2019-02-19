package graphql

import (
	"context"
	"net/http"
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

// NetworkMiddleware are functions can be passed to NetworkQueryer.WithMiddleware to affect its internal
// behavior
type NetworkMiddleware func(*http.Request) error

// QueryerWithMiddlewares is an interface for queryers that support network middlewares
type QueryerWithMiddlewares interface {
	WithMiddlewares(wares []NetworkMiddleware) Queryer
}

// HTTPQueryer is an interface for queryers that let you configure an underlying http.Client
type HTTPQueryer interface {
	WithHTTPClient(client *http.Client) Queryer
}

// HTTPQueryerWithMiddlewares is an interface for queryers that let you configure an underlying http.Client
// and accept middlewares
type HTTPQueryerWithMiddlewares interface {
	WithHTTPClient(client *http.Client) Queryer
	WithMiddlewares(wares []NetworkMiddleware) Queryer
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
