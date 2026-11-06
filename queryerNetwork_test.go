package graphql

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSingleRequestQueryer(t *testing.T) {
	t.Parallel()
	const someURL = "foo"
	assert.Equal(t, &SingleRequestQueryer{
		queryer: &NetworkQueryer{URL: someURL},
	}, NewSingleRequestQueryer(someURL))
}

func TestSingleRequestQueryer_query_results(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description        string
		responseStatusCode int
		responseBody       string
		response           any
		expectResponse     any
		expectError        error
		expectErrorType    error
	}{
		{
			description:        "successful with map response type",
			response:           new(map[string]any),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				}
			}`,
			expectResponse: &map[string]any{
				"foo": "bar",
			},
		},
		{
			description: "successful with unknown response type",
			response: new(struct {
				Foo string
			}),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				}
			}`,
			expectResponse: &struct {
				Foo string
			}{
				Foo: "bar",
			},
		},
		{
			description:        "successful with QueryResult type",
			response:           new(QueryResult),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				},
				"extensions": {
					"baz": "biff"
				}
			}`,
			expectResponse: &QueryResult{
				Data: map[string]any{
					"foo": "bar",
				},
				Extensions: map[string]any{
					"baz": "biff",
				},
			},
		},
		{
			description:        "partial success with map response type",
			response:           new(map[string]any),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				},
				"errors": [
					{"message": "baz"}
				]
			}`,
			expectResponse: &map[string]any{
				"foo": "bar",
			},
			expectError: ErrorList{
				&Error{Message: "baz"},
			},
		},
		{
			description: "partial success with unknown response type",
			response: new(struct {
				Foo string
			}),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				},
				"errors": [
					{"message": "baz"}
				]
			}`,
			expectResponse: &struct {
				Foo string
			}{
				Foo: "bar",
			},
			expectError: ErrorList{
				&Error{Message: "baz"},
			},
		},
		{
			description:        "partial success with QueryResult response type",
			response:           new(QueryResult),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				},
				"errors": [
					{"message": "baz"}
				],
				"extensions": {
					"biff": "boo"
				}
			}`,
			expectResponse: &QueryResult{
				Data: map[string]any{
					"foo": "bar",
				},
				Errors: new([]*Error{
					{Message: "baz"},
				}),
				Extensions: map[string]any{
					"biff": "boo",
				},
			},
			expectError: ErrorList{
				&Error{Message: "baz"},
			},
		},
		{
			description:        "invalid response type",
			response:           "not a pointer",
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				}
			}`,
			expectResponse: "not a pointer",
			expectError:    errors.New("result must be a pointer"),
		},
		{
			description:        "fail to decode",
			response:           new(chan string),
			responseStatusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"foo": "bar"
				}
			}`,
			expectResponse:  new(chan string),
			expectErrorType: new(mapstructure.DecodeError{}),
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			const (
				someQuery = `{"query":"foo","operationName":"","variables":null}`
				someURL   = "foo"
			)
			var query QueryInput
			require.NoError(t, json.Unmarshal([]byte(someQuery), &query))
			queryer := NewSingleRequestQueryer(someURL).
				WithHTTPClient(testHTTPClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					assert.JSONEq(t, someQuery, string(body))

					w.WriteHeader(tc.responseStatusCode)
					fmt.Fprint(w, tc.responseBody)
				})))

			err := queryer.Query(t.Context(), &query, tc.response)
			assert.Equal(t, tc.expectResponse, tc.response)
			if tc.expectErrorType != nil {
				assert.ErrorAs(t, err, &tc.expectErrorType)
			} else {
				assert.Equal(t, tc.expectError, err)
			}
		})
	}
}
