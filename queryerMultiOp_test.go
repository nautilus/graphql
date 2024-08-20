package graphql

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type contextKey int

const (
	requestLabel contextKey = iota
	responseCount
)

func TestNewMultiOpQueryer(t *testing.T) {
	queryer := NewMultiOpQueryer("foo", 1*time.Millisecond, 100)

	// make sure the queryer config is all correct
	assert.Equal(t, "foo", queryer.queryer.URL)
	assert.Equal(t, 1*time.Millisecond, queryer.BatchInterval)
	assert.Equal(t, 100, queryer.MaxBatchSize)
}

func TestMultiOpQueryer_batchesRequests(t *testing.T) {
	nCalled := 0

	// the bundle time of the queryer
	interval := 10 * time.Millisecond

	// create a queryer that we will use that has a client that keeps track of the
	// number of times it was called
	queryer := NewMultiOpQueryer("foo", interval, 100).WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			nCalled++

			label := req.Context().Value(requestLabel).(string)

			body := ""
			for i := 0; i < req.Context().Value(responseCount).(int); i++ {
				body += fmt.Sprintf(`{ "data": { "nCalled": "%s:%v" } },`, label, nCalled)
			}

			return &http.Response{
				StatusCode: 200,
				// Send response to be tested
				Body: ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf(`[
					%s
				]`, body[:len(body)-1]))),
				// Must be set to non-nil value or it panics
				Header: make(http.Header),
			}
		}),
	})

	// the query we will be batching
	query := "{ nCalled }"

	// places to hold the results
	result1 := map[string]interface{}{}
	result2 := map[string]interface{}{}
	result3 := map[string]interface{}{}

	// query once on its own
	ctx1 := context.WithValue(context.WithValue(context.Background(), requestLabel, "1"), responseCount, 1)
	queryer.Query(ctx1, &QueryInput{Query: query}, &result1)

	// wait a bit
	time.Sleep(interval + 10*time.Millisecond)

	// query twice back to back
	count := &sync.WaitGroup{}
	count.Add(1)
	go func() {
		ctx2 := context.WithValue(context.WithValue(context.Background(), requestLabel, "2"), responseCount, 2)
		queryer.Query(ctx2, &QueryInput{Query: query}, &result2)
		count.Done()
	}()
	count.Add(1)
	go func() {
		ctx3 := context.WithValue(context.WithValue(context.Background(), requestLabel, "2"), responseCount, 2)
		queryer.Query(ctx3, &QueryInput{Query: query}, &result3)
		count.Done()
	}()

	// wait for the queries to be done
	count.Wait()

	// make sure that we only invoked the client twice
	assert.Equal(t, 2, nCalled)

	// make sure that we got the right results
	assert.Equal(t, map[string]interface{}{"nCalled": "1:1"}, result1)
	assert.Equal(t, map[string]interface{}{"nCalled": "2:2"}, result2)
	assert.Equal(t, map[string]interface{}{"nCalled": "2:2"}, result3)
}

func TestMultiOpQueryer_partial_success(t *testing.T) {
	t.Parallel()
	queryer := NewMultiOpQueryer("someURL", 1*time.Millisecond, 10).WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) *http.Response {
			w := httptest.NewRecorder()
			fmt.Fprint(w, `
				[
					{
						"data": {
							"foo": "bar"
						},
						"errors": [
							{"message": "baz"}
						]
					}
				]
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
