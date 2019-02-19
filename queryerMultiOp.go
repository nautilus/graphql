package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mitchellh/mapstructure"
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
	// the payload
	payload, err := json.Marshal(map[string]interface{}{
		"query":         input.Query,
		"variables":     input.Variables,
		"operationName": input.OperationName,
	})
	if err != nil {
		return err
	}

	// construct the initial request we will send to the client
	req, err := http.NewRequest("POST", q.URL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	// add the current context to the request
	acc := req.WithContext(ctx)
	acc.Header.Set("Content-Type", "application/json")

	// we could have any number of middlewares that we have to go through so
	for _, mware := range q.middlewares {
		err := mware(acc)
		if err != nil {
			return err
		}
	}

	// fire the response to the queryer's url
	resp, err := q.client.Do(acc)
	if err != nil {
		return err
	}

	// read the full body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	result := map[string]interface{}{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("Response body was not valid json: %s", string(body))
	}

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
