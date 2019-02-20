package graphql

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mitchellh/mapstructure"
)

// SingleRequestQueryer sends the query to a url and returns the response
type SingleRequestQueryer struct {
	// internals for bundling queries
	queryer *NetworkQueryer
}

// NewSingleRequestQueryer returns a SingleRequestQueryer pointed to the given url
func NewSingleRequestQueryer(url string) *SingleRequestQueryer {
	return &SingleRequestQueryer{
		queryer: &NetworkQueryer{URL: url},
	}
}

// WithMiddlewares returns a network queryer that will apply the provided middlewares
func (q *SingleRequestQueryer) WithMiddlewares(mwares []NetworkMiddleware) Queryer {
	// for now just change the internal reference
	q.queryer.Middlewares = mwares

	// return it
	return q
}

// WithHTTPClient lets the user configure the underlying http client being used
func (q *SingleRequestQueryer) WithHTTPClient(client *http.Client) Queryer {
	q.queryer.Client = client

	return q
}

func (q *SingleRequestQueryer) URL() string {
	return q.queryer.URL
}

// Query sends the query to the designated url and returns the response.
func (q *SingleRequestQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
	// the payload
	payload, err := json.Marshal(map[string]interface{}{
		"query":         input.Query,
		"variables":     input.Variables,
		"operationName": input.OperationName,
	})
	if err != nil {
		return err
	}

	// send that query to the api and write the appropriate response to the receiver
	response, err := q.queryer.SendQuery(ctx, payload)
	if err != nil {
		return err
	}

	result := map[string]interface{}{}
	if err = json.Unmarshal(response, &result); err != nil {
		return err
	}

	// otherwise we have to copy the response onto the receiver
	if err = q.queryer.ExtractErrors(result); err != nil {
		return err
	}

	// assign the result under the data key to the receiver
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  receiver,
	})
	if err != nil {
		return err
	}

	// the only way for things to go wrong now happen while decoding
	return decoder.Decode(result["data"])
}
