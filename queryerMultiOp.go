package graphql

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
)

// MultiOpQueryer is a queryer that will batch subsequent query on some interval into a single network request
// to a single target
type MultiOpQueryer struct {
	MaxBatchSize  int
	BatchInterval time.Duration
	URL           string

	// the http client to use for requests
	client *http.Client
	// the list of middlewares to apply to the request object
	middlewares []NetworkMiddleware

	// some internal attributes to track bundling
	bundleChan    chan *newOperation
	bundleLock    *sync.Mutex
	bundlePending bool
}

type newOperation struct {
	Input      *QueryInput
	ResponseCh chan map[string]interface{}
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

// Query bundles queries that happen within the given interval into a single network request
// whose body is a list of the operation payload.
func (q *MultiOpQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	// create a channel where we will get the response
	responseCh := make(chan map[string]interface{})

	// add this query to the bundle
	q.bundleChan <- &newOperation{
		Input:      input,
		ResponseCh: responseCh,
	}

	// make sure we have access to the lock
	q.bundleLock.Lock()
	// if this is the first query since the last time we sent off a bundle
	if !q.bundlePending {
		// we have to start a goroutine that will fulfill the pending requests
		go q.waitThenDrain()

		// we now have a bundle pending
		q.bundlePending = true
	}
	q.bundleLock.Unlock()

	// wait for the result
	result := <-responseCh

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

			errList = append(errList, NewError("", message))
		}

		return errList
	}

	// assign the result under the data key to the receiver
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  receiver,
	})
	if err != nil {
		return err
	}

	err = decoder.Decode(result["data"])
	if err != nil {
		return err
	}

	// pass the result along
	return nil
}

// wait then drain is called whenever a new bundle has to be kicked off
func (q *MultiOpQueryer) waitThenDrain() {
	// the first thing we have to do is wait the designated amount of time
	time.Sleep(q.BatchInterval)

}
