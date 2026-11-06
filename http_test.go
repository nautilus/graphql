package graphql

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPClient(tb testing.TB, handler http.Handler) *http.Client {
	return testHTTPClientWithNetworkError(tb, handler, nil)
}

func testHTTPClientWithNetworkError(tb testing.TB, handler http.Handler, networkErr error) *http.Client {
	tb.Helper()
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			return resp.Result(), nil
		}),
	}
}
