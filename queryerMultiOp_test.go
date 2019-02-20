package graphql

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type contextKey int

const requestLabel contextKey = iota

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

			return &http.Response{
				StatusCode: 200,
				// Send response to be tested
				Body: ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf("[{ \"nCalled\": \"%v:%v\" }]", label, nCalled))),
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

	// invoke the queries as close together as possible
	err1 := queryer.Query(context.WithValue(context.Background(), requestLabel, "1"), &QueryInput{Query: query}, &result1)

	// wait a little longer than the batch interval
	time.Sleep(interval + 10*time.Millisecond)

	// invoke the queryer in quick sucession
	err2 := queryer.Query(context.WithValue(context.Background(), requestLabel, "2"), &QueryInput{Query: query}, &result2)
	err3 := queryer.Query(context.WithValue(context.Background(), requestLabel, "3"), &QueryInput{Query: query}, &result3)

	if !assert.Nil(t, err1) || !assert.Nil(t, err2) || !assert.Nil(t, err3) {
		return
	}

	// make sure that we only invoked the client twice
	assert.Equal(t, 2, nCalled)

	// make sure that we got the right results
	assert.Equal(t, map[string]interface{}{"nCalled": 1}, result1)
	assert.Equal(t, map[string]interface{}{"nCalled": 2}, result2)
	assert.Equal(t, map[string]interface{}{"nCalled": 3}, result3)
}
