package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type responseMiddlewareCtxKey struct{}

// respondWith returns an http.Client whose every request is answered with the given body
func respondWith(body string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(*http.Request) *http.Response {
			w := httptest.NewRecorder()
			fmt.Fprint(w, body)
			return w.Result()
		}),
	}
}

func TestSingleRequestQueryer_responseMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("sees the full response before data is decoded", func(t *testing.T) {
		t.Parallel()
		var seen map[string]interface{}
		var seenCtx context.Context

		queryer := NewSingleRequestQueryer("someURL")
		queryer.WithHTTPClient(respondWith(`{ "data": { "foo": "bar" }, "extensions": { "hello": "world" } }`))
		queryer.WithResponseMiddlewares([]ResponseMiddleware{
			func(ctx context.Context, response map[string]interface{}) error {
				seen = response
				seenCtx = ctx
				return nil
			},
		})

		ctx := context.WithValue(context.Background(), responseMiddlewareCtxKey{}, "value")
		var result map[string]interface{}
		err := queryer.Query(ctx, &QueryInput{Query: "{ foo }"}, &result)

		assert.NoError(t, err, "a middleware that returns nil must not affect the query")
		assert.Equal(t, map[string]interface{}{"foo": "bar"}, result,
			"the receiver still gets only the data object, exactly as without middlewares")
		assert.Equal(t, map[string]interface{}{
			"data":       map[string]interface{}{"foo": "bar"},
			"extensions": map[string]interface{}{"hello": "world"},
		}, seen, "the middleware sees the whole parsed response, including keys the receiver never gets (extensions)")
		assert.Equal(t, "value", seenCtx.Value(responseMiddlewareCtxKey{}),
			"the middleware runs with the context passed to Query, so callers can hand it per-query state")
	})

	t.Run("runs for multipart requests too", func(t *testing.T) {
		t.Parallel()
		var seen map[string]interface{}
		var contentType string

		queryer := NewSingleRequestQueryer("someURL")
		queryer.WithHTTPClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) *http.Response {
				contentType = req.Header.Get("Content-Type")
				w := httptest.NewRecorder()
				fmt.Fprint(w, `{ "data": { "upload": "ok" }, "extensions": { "hello": "world" } }`)
				return w.Result()
			}),
		})
		queryer.WithResponseMiddlewares([]ResponseMiddleware{
			func(ctx context.Context, response map[string]interface{}) error {
				seen = response
				return nil
			},
		})

		var result map[string]interface{}
		err := queryer.Query(context.Background(), &QueryInput{
			Query:     "mutation($file: Upload!) { upload(file: $file) }",
			Variables: map[string]interface{}{"file": Upload{File: io.NopCloser(strings.NewReader("content")), FileName: "file.txt"}},
		}, &result)

		assert.NoError(t, err, "a middleware that returns nil must not affect an upload query")
		assert.True(t, strings.HasPrefix(contentType, "multipart/form-data"),
			"an Upload variable must have switched the request to multipart, otherwise this test exercises the JSON branch; got %q", contentType)
		assert.Equal(t, map[string]interface{}{"upload": "ok"}, result,
			"the receiver still gets only the data object on the multipart branch")
		assert.Equal(t, map[string]interface{}{"hello": "world"}, seen["extensions"],
			"the middleware also runs on the multipart branch and sees that response's extensions")
	})

	t.Run("an error aborts the query", func(t *testing.T) {
		t.Parallel()
		queryer := NewSingleRequestQueryer("someURL")
		queryer.WithHTTPClient(respondWith(`{ "data": { "foo": "bar" } }`))
		queryer.WithResponseMiddlewares([]ResponseMiddleware{
			func(context.Context, map[string]interface{}) error { return errors.New("rejected") },
		})

		var result map[string]interface{}
		err := queryer.Query(context.Background(), &QueryInput{Query: "{ foo }"}, &result)

		assert.EqualError(t, err, "rejected", "the middleware's error is returned from Query unchanged")
		assert.Nil(t, result, "when a middleware fails, nothing is decoded into the receiver")
	})
}

func TestMultiOpQueryer_responseMiddlewares(t *testing.T) {
	t.Parallel()
	interval := 10 * time.Millisecond

	// every bundled query gets its own entry in the batched response, labelled by its position
	queryer := NewMultiOpQueryer("someURL", interval, 100)
	queryer.WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			var ops []map[string]interface{}
			body, _ := io.ReadAll(req.Body)
			assert.NoError(t, json.Unmarshal(body, &ops), "the batching queryer must send a JSON array of operations")

			entries := make([]string, 0, len(ops))
			for _, op := range ops {
				// the query text names the operation, e.g. "{ a }"
				name := strings.Trim(op["query"].(string), "{ }")
				entries = append(entries, fmt.Sprintf(`{ "data": { "name": "%s" }, "extensions": { "for": "%s" } }`, name, name))
			}
			w := httptest.NewRecorder()
			fmt.Fprint(w, "["+strings.Join(entries, ",")+"]")
			return w.Result()
		}),
	})

	// record which context saw which response
	var mu sync.Mutex
	seen := map[string]interface{}{}
	queryer.WithResponseMiddlewares([]ResponseMiddleware{
		func(ctx context.Context, response map[string]interface{}) error {
			mu.Lock()
			defer mu.Unlock()
			seen[ctx.Value(responseMiddlewareCtxKey{}).(string)] = response["extensions"]
			return nil
		},
	})

	// fire two queries that will be bundled into one request
	var wg sync.WaitGroup
	for _, name := range []string{"a", "b"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), responseMiddlewareCtxKey{}, name)
			var result map[string]interface{}
			err := queryer.Query(ctx, &QueryInput{Query: "{ " + name + " }"}, &result)
			assert.NoError(t, err, "query %s: a middleware that returns nil must not affect a bundled query", name)
			assert.Equal(t, map[string]interface{}{"name": name}, result,
				"query %s: the receiver gets the data of this query's own entry in the batched response", name)
		}(name)
	}
	wg.Wait()

	// each query's middleware saw that query's own entry
	assert.Equal(t, map[string]interface{}{
		"a": map[string]interface{}{"for": "a"},
		"b": map[string]interface{}{"for": "b"},
	}, seen, "the middleware ran once per bundled query, each time with that query's context and that query's own entry of the batched response, never another query's")
}
