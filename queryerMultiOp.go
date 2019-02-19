package graphql

import (
	"context"
	"net/http"
	"time"
)

// MultiOpQueryer is a queryer that will batch subsequent query on some interval into a single network request
// to a single target
type MultiOpQueryer struct {
	MaxBatchSize  int
	BatchInterval time.Duration
	URL           string

	client      *http.Client
	middlewares []NetworkMiddleware
}

// NewMultiOpQueryer returns a MultiOpQueryer with the provided paramters
func NewMultiOpQueryer(url string, interval time.Duration, maxBatchSize int) *MultiOpQueryer {
	return &MultiOpQueryer{
		MaxBatchSize:  maxBatchSize,
		BatchInterval: interval,
		URL:           url,
	}
}

// WithMiddlewares lets the user assign middlewares to the queryer
func (q *MultiOpQueryer) WithMiddlewares(mwares []NetworkMiddleware) Queryer {
	q.middlewares = mwares
	return q
}

// WithHTTPClient lets the user configure the client to use when making network requests
func (q *MultiOpQueryer) WithHTTPClient(client *http.Client) Queryer {
	q.client = client
	return q
}

// Query bundles queries that happen Swithin the given interval into a single network request
// whose body is a list of the operation payload.
func (q *MultiOpQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	return nil
}
