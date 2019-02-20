package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/graph-gophers/dataloader"
)

// MultiOpQueryer is a queryer that will batch subsequent query on some interval into a single network request
// to a single target
type MultiOpQueryer struct {
	MaxBatchSize  int
	BatchInterval time.Duration

	// internals for bundling queries
	queryer *NetworkQueryer
	loader  *dataloader.Loader
}

// NewMultiOpQueryer returns a MultiOpQueryer with the provided paramters
func NewMultiOpQueryer(url string, interval time.Duration, maxBatchSize int) *MultiOpQueryer {
	queryer := &MultiOpQueryer{
		MaxBatchSize:  maxBatchSize,
		BatchInterval: interval,
	}

	// instantiate a dataloader we can use for queries
	queryer.loader = dataloader.NewBatchedLoader(queryer.loadQuery, dataloader.WithCache(&dataloader.NoCache{}))

	// instantiate a network queryer we can use later
	queryer.queryer = &NetworkQueryer{
		URL: url,
	}

	return queryer
}

// WithMiddlewares lets the user assign middlewares to the queryer
func (q *MultiOpQueryer) WithMiddlewares(mwares []NetworkMiddleware) Queryer {
	q.queryer.Middlewares = mwares
	return q
}

// WithHTTPClient lets the user configure the client to use when making network requests
func (q *MultiOpQueryer) WithHTTPClient(client *http.Client) Queryer {
	q.queryer.Client = client
	return q
}

// Query bundles queries that happen within the given interval into a single network request
// whose body is a list of the operation payload.
func (q *MultiOpQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	// process the input
	result, err := q.loader.Load(ctx, input)()
	if err != nil {
		return err
	}

	// we need to take the result and set the receiver to match (keep errors in mind too - extractErrors)

	// format the result as needed
	return q.queryer.ExtractErrors(result.(map[string]interface{}))
}

func (q *MultiOpQueryer) loadQuery(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	// a place to store the results
	results := []*dataloader.Result{}

	// the keys serialize to the correct representation
	payload, err := json.Marshal(keys)
	if err != nil {
		// we need to result the same error for each result
		for range keys {
			results = append(results, &dataloader.Result{Error: err})
		}
		return results
	}

	// send the payload to the server
	response, err := q.queryer.SendQuery(ctx, payload)
	if err != nil {
		// we need to result the same error for each result
		for range keys {
			results = append(results, &dataloader.Result{Error: err})
		}
		return results
	}

	// a place to handle each result
	queryResults := []map[string]interface{}{}
	err = json.Unmarshal(response, &queryResults)
	if err != nil {
		// we need to result the same error for each result
		for range keys {
			results = append(results, &dataloader.Result{Error: err})
		}
		return results
	}

	// take the result from the query and turn it into something dataloader is okay with
	for _, result := range queryResults {
		results = append(results, &dataloader.Result{Data: result})
	}

	// return the results
	return results
}
