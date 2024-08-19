package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"
)

// RemoteSchema encapsulates a particular schema that can be executed by sending network requests to the
// specified URL.
type RemoteSchema struct {
	Schema *ast.Schema
	URL    string
}

// QueryInput provides all of the information required to fire a query
type QueryInput struct {
	Query         string                 `json:"query"`
	QueryDocument *ast.QueryDocument     `json:"-"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// String returns a guaranteed unique string that can be used to identify the input
func (i *QueryInput) String() string {
	// let's just marshal the input
	marshaled, err := json.Marshal(i)
	if err != nil {
		return ""
	}

	// return the result
	return string(marshaled)
}

// Raw returns the "raw underlying value of the key" when used by dataloader
func (i *QueryInput) Raw() interface{} {
	return i
}

// Queryer is a interface for objects that can perform
type Queryer interface {
	Query(context.Context, *QueryInput, interface{}) error
}

// NetworkMiddleware are functions can be passed to SingleRequestQueryer.WithMiddleware to affect its internal
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

type NetworkQueryer struct {
	URL         string
	Middlewares []NetworkMiddleware
	Client      *http.Client
}

// SendQuery is responsible for sending the provided payload to the desingated URL
func (q *NetworkQueryer) SendQuery(ctx context.Context, payload []byte) ([]byte, error) {
	// construct the initial request we will send to the client
	req, err := http.NewRequest("POST", q.URL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	// add the current context to the request
	acc := req.WithContext(ctx)
	acc.Header.Set("Content-Type", "application/json")

	return q.sendRequest(acc)
}

// SendMultipart is responsible for sending multipart request to the desingated URL
func (q *NetworkQueryer) SendMultipart(ctx context.Context, payload []byte, contentType string) ([]byte, error) {
	// construct the initial request we will send to the client
	req, err := http.NewRequest("POST", q.URL, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	// add the current context to the request
	acc := req.WithContext(ctx)
	acc.Header.Set("Content-Type", contentType)

	return q.sendRequest(acc)
}

func (q *NetworkQueryer) sendRequest(acc *http.Request) ([]byte, error) {
	// we could have any number of middlewares that we have to go through so
	for _, mware := range q.Middlewares {
		err := mware(acc)
		if err != nil {
			return nil, err
		}
	}

	// fire the response to the queryer's url
	if q.Client == nil {
		q.Client = &http.Client{}
	}

	resp, err := q.Client.Do(acc)
	if err != nil {
		return nil, err
	}

	// read the full body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return body, errors.New("response was not successful with status code: " + strconv.Itoa(resp.StatusCode))
	}

	// we're done
	return body, err
}

// ExtractErrors takes the result from a remote query and writes it to the provided pointer
func (q *NetworkQueryer) ExtractErrors(result map[string]interface{}) error {
	// if there is an error
	if _, ok := result["errors"]; ok {
		// a list of errors from the response
		errList := ErrorList{}

		// build up a list of errors
		errs, ok := result["errors"].([]interface{})
		if !ok {
			return errors.New("errors was not a list")
		}

		// a list of error messages
		for _, err := range errs {
			obj, ok := err.(map[string]interface{})
			if !ok {
				return errors.New("encountered non-object error")
			}

			message, ok := obj["message"].(string)
			if !ok {
				return errors.New("error message was not a string")
			}

			var extensions map[string]interface{}
			if e, ok := obj["extensions"].(map[string]interface{}); ok {
				extensions = e
			}

			var path []interface{}
			if p, ok := obj["path"].([]interface{}); ok {
				path = p
			}

			errList = append(errList, &Error{
				Message:    message,
				Path:       path,
				Extensions: extensions,
			})
		}

		return errList
	}

	// pass the result along
	return nil
}
