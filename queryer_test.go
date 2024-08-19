package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestQueryerFunc_success(t *testing.T) {
	expected := map[string]interface{}{"hello": "world"}

	queryer := QueryerFunc(
		func(*QueryInput) (interface{}, error) {
			return expected, nil
		},
	)

	// a place to write the result
	result := map[string]interface{}{}

	err := queryer.Query(context.Background(), &QueryInput{}, &result)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// make sure we copied the right result
	assert.Equal(t, expected, result)
}

func TestQueryerFunc_failure(t *testing.T) {
	expected := errors.New("message")

	queryer := QueryerFunc(
		func(*QueryInput) (interface{}, error) {
			return nil, expected
		},
	)

	err := queryer.Query(context.Background(), &QueryInput{}, &map[string]interface{}{})

	// make sure we got the right error
	assert.Equal(t, expected, err)
}

func TestQueryerFunc_partial_success(t *testing.T) {
	t.Parallel()
	someData := map[string]interface{}{"foo": "bar"}
	someError := errors.New("baz")

	queryer := QueryerFunc(func(*QueryInput) (interface{}, error) {
		return someData, someError
	})

	result := map[string]interface{}{}

	err := queryer.Query(context.Background(), &QueryInput{}, &result)
	assert.ErrorIs(t, err, someError)
	assert.Equal(t, someData, result)
}

func TestHTTPQueryerBasicCases(t *testing.T) {
	// this test run a suite of tests for every queryer in the table
	queryerTable := []struct {
		name       string
		queryer    HTTPQueryer
		wrapInList bool
	}{
		{
			"Single Request",
			NewSingleRequestQueryer("hello"),
			false,
		},
		{
			"MultiOp",
			NewMultiOpQueryer("hello", 1*time.Millisecond, 10),
			true,
		},
	}

	// for each queryer we have to test
	for _, row := range queryerTable {
		t.Run(row.name, func(t *testing.T) {
			t.Run("Sends Queries", func(t *testing.T) {
				// build a query to test should be equivalent to
				// targetQueryBody := `
				// 	{
				// 		hello(world: "hello") {
				// 			world
				// 		}
				// 	}
				// `

				// the result we expect back
				expected := map[string]interface{}{
					"foo": "bar",
				}

				// the corresponding query document
				query := `
					{
						hello(world: "hello") {
							world
						}
					}
				`

				httpQueryer := row.queryer.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(req *http.Request) *http.Response {
						var result interface{}
						if row.wrapInList {
							result = []map[string]interface{}{{"data": expected}}
						} else {
							result = map[string]interface{}{"data": expected}
						}

						// serialize the json we want to send back
						marshaled, err := json.Marshal(result)
						// if something went wrong
						if err != nil {
							return &http.Response{
								StatusCode: 500,
								Body:       ioutil.NopCloser(bytes.NewBufferString("Something went wrong")),
								Header:     make(http.Header),
							}
						}

						return &http.Response{
							StatusCode: 200,
							// Send response to be tested
							Body: ioutil.NopCloser(bytes.NewBuffer(marshaled)),
							// Must be set to non-nil value or it panics
							Header: make(http.Header),
						}
					}),
				})

				// get the response of the query
				result := map[string]interface{}{}
				err := httpQueryer.Query(context.Background(), &QueryInput{Query: query}, &result)
				if err != nil {
					t.Error(err)
					return
				}
				if result == nil {
					t.Error("Did not get a result back")
					return
				}

				// make sure we got what we expected
				assert.Equal(t, expected, result)
			})

			t.Run("Handles error response", func(t *testing.T) {
				// the table for the tests
				for _, errorRow := range []struct {
					Message    string
					ErrorShape interface{}
				}{
					{
						"Well Structured Error",
						[]map[string]interface{}{
							{
								"message": "message",
							},
						},
					},
					{
						"Errors Not Lists",
						map[string]interface{}{
							"message": "message",
						},
					},
					{
						"Errors Lists of Not Strings",
						[]string{"hello"},
					},
					{
						"Errors No messages",
						[]map[string]interface{}{},
					},
					{
						"Message not string",
						[]map[string]interface{}{
							{
								"message": true,
							},
						},
					},
					{
						"No Errors",
						nil,
					},
				} {
					t.Run(errorRow.Message, func(t *testing.T) {
						// the corresponding query document
						query := `
							{
								hello(world: "hello") {
									world
								}
							}
						`

						queryer := row.queryer.WithHTTPClient(&http.Client{
							Transport: roundTripFunc(func(req *http.Request) *http.Response {
								response := map[string]interface{}{
									"data": nil,
								}

								// if we are supposed to have an error
								if errorRow.ErrorShape != nil {
									response["errors"] = errorRow.ErrorShape
								}

								var finalResponse interface{} = response
								if row.wrapInList {
									finalResponse = []map[string]interface{}{response}
								}

								// serialize the json we want to send back
								result, err := json.Marshal(finalResponse)
								// if something went wrong
								if err != nil {
									return &http.Response{
										StatusCode: 500,
										Body:       ioutil.NopCloser(bytes.NewBufferString("Something went wrong")),
										Header:     make(http.Header),
									}
								}

								return &http.Response{
									StatusCode: 200,
									// Send response to be tested
									Body: ioutil.NopCloser(bytes.NewBuffer(result)),
									// Must be set to non-nil value or it panics
									Header: make(http.Header),
								}
							}),
						})

						// get the response of the query
						result := map[string]interface{}{}
						err := queryer.Query(context.Background(), &QueryInput{Query: query}, &result)

						// if we're supposed to hav ean error
						if errorRow.ErrorShape != nil {
							assert.NotNil(t, err)
						} else {
							assert.Nil(t, err)
						}
					})
				}
			})

			t.Run("Error Lists", func(t *testing.T) {
				// the corresponding query document
				query := `
					{
						hello(world: "hello") {
							world
						}
					}
				`

				queryer := row.queryer.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(req *http.Request) *http.Response {
						response := `{
							"data": null,
							"errors": [
								{"message":"hello"}
							]
						}`
						if row.wrapInList {
							response = fmt.Sprintf("[%s]", response)
						}

						return &http.Response{
							StatusCode: 200,
							// Send response to be tested
							Body: ioutil.NopCloser(bytes.NewBuffer([]byte(response))),
							// Must be set to non-nil value or it panics
							Header: make(http.Header),
						}
					}),
				})

				// get the error of the query
				err := queryer.Query(context.Background(), &QueryInput{Query: query}, &map[string]interface{}{})
				// if we didn't get an error at all
				if err == nil {
					t.Error("Did not encounter an error")
					return
				}

				_, ok := err.(ErrorList)
				if !ok {
					t.Errorf("response of queryer was not an error list: %v", err.Error())
					return
				}
			})

			t.Run("Responds with Error", func(t *testing.T) {
				// the corresponding query document
				query := `
					{
						hello
					}
				`

				queryer := row.queryer.WithHTTPClient(&http.Client{
					Transport: roundTripFunc(func(req *http.Request) *http.Response {
						// send an error back
						return &http.Response{
							StatusCode: 500,
							Body:       ioutil.NopCloser(bytes.NewBufferString("Something went wrong")),
							Header:     make(http.Header),
						}
					}),
				})

				// get the response of the query
				var result interface{}
				err := queryer.Query(context.Background(), &QueryInput{Query: query}, result)
				if err == nil {
					t.Error("Did not receive an error")
					return
				}
			})
		})
	}
}

func TestQueryerWithMiddlewares(t *testing.T) {
	queryerTable := []struct {
		name       string
		queryer    HTTPQueryerWithMiddlewares
		wrapInList bool
	}{
		{
			"Single Request",
			NewSingleRequestQueryer("hello"),
			false,
		},
		{
			"MultiOp",
			NewMultiOpQueryer("hello", 1*time.Millisecond, 10),
			true,
		},
	}

	for _, row := range queryerTable {
		t.Run(row.name, func(t *testing.T) {
			t.Run("Middleware Failures", func(t *testing.T) {
				queryer := row.queryer.WithMiddlewares([]NetworkMiddleware{
					func(r *http.Request) error {
						return errors.New("This One")
					},
				})

				// the input to the query
				input := &QueryInput{
					Query: "",
				}

				// fire the query
				err := queryer.Query(context.Background(), input, &map[string]interface{}{})
				if err == nil {
					t.Error("Did not enounter an error when we should have")
					return
				}
				if err.Error() != "This One" {
					t.Errorf("Did not encountered expected error message: Expected 'This One', found %v", err.Error())
				}
			})

			t.Run("Middlware success", func(t *testing.T) {
				queryer := row.queryer.WithMiddlewares([]NetworkMiddleware{
					func(r *http.Request) error {
						r.Header.Set("Hello", "World")

						return nil
					},
				})

				if q, ok := queryer.(HTTPQueryerWithMiddlewares); ok {
					queryer = q.WithHTTPClient(&http.Client{
						Transport: roundTripFunc(func(req *http.Request) *http.Response {
							// if we did not get the right header value
							if req.Header.Get("Hello") != "World" {
								return &http.Response{
									StatusCode: http.StatusExpectationFailed,
									// Send response to be tested
									Body: ioutil.NopCloser(bytes.NewBufferString("Did not receive the right header")),
									// Must be set to non-nil value or it panics
									Header: make(http.Header),
								}
							}

							// serialize the json we want to send back
							result, _ := json.Marshal(map[string]interface{}{
								"allUsers": []string{
									"John Jacob",
									"Jinglehymer Schmidt",
								},
							})
							if row.wrapInList {
								result = []byte(fmt.Sprintf("[%s]", string(result)))
							}

							return &http.Response{
								StatusCode: 200,
								// Send response to be tested
								Body: ioutil.NopCloser(bytes.NewBuffer(result)),
								// Must be set to non-nil value or it panics
								Header: make(http.Header),
							}
						}),
					})
				}

				// the input to the query
				input := &QueryInput{
					Query: "",
				}

				err := queryer.Query(context.Background(), input, &map[string]interface{}{})
				if err != nil {
					t.Error(err.Error())
					return
				}
			})
		})
	}
}

func TestNetworkQueryer_partial_success(t *testing.T) {
	t.Parallel()
	queryer := NewSingleRequestQueryer("someURL").WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) *http.Response {
			w := httptest.NewRecorder()
			fmt.Fprint(w, `
				{
					"data": {
						"foo": "bar"
					},
					"errors": [
						{"message": "baz"}
					]
				}
			`)
			return w.Result()
		}),
	})
	var result any
	err := queryer.Query(context.Background(), &QueryInput{Query: "query { hello }"}, &result)
	assert.Equal(t, map[string]any{
		"foo": "bar",
	}, result)
	assert.EqualError(t, err, "baz")
}
