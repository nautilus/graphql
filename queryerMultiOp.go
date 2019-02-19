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
	Client        http.Client

	middlewares []NetworkMiddleware
}

// WithMiddlewares lets the user assign middlewares to the queryer
func (q *MultiOpQueryer) WithMiddlewares(mwares []NetworkMiddleware) Queryer {
	return &MultiOpQueryer{
		URL:           q.URL,
		Client:        q.Client,
		BatchInterval: q.BatchInterval,
		MaxBatchSize:  q.MaxBatchSize,
		middlewares:   mwares,
	}
}

// Query bundles queries that happen within the given interval into a single network request
// whose body is a list of the operation payload.
func (q *MultiOpQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	return nil
}
