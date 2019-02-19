package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mitchellh/mapstructure"
)

// NetworkQueryer sends the query to a url and returns the response
type NetworkQueryer struct {
	URL         string
	client      *http.Client
	middlewares []NetworkMiddleware
}

// NewNetworkQueryer returns a NetworkQueryer pointed to the given url
func NewNetworkQueryer(url string) *NetworkQueryer {
	return &NetworkQueryer{
		URL: url,
	}
}

// WithMiddlewares returns a network queryer that will apply the provided middlewares
func (q *NetworkQueryer) WithMiddlewares(mwares []NetworkMiddleware) Queryer {
	// for now just change the internal reference
	q.middlewares = mwares

	// return it
	return q
}

// WithHTTPClient lets the user configure the underlying http client being used
func (q *NetworkQueryer) WithHTTPClient(client *http.Client) Queryer {
	q.client = client

	return q
}

// Query sends the query to the designated url and returns the response.
func (q *NetworkQueryer) Query(ctx context.Context, input *QueryInput, receiver interface{}) error {
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
	req.Header.Set("Content-Type", "application/json")

	// we could have any number of middlewares that we have to go through so
	for _, mware := range q.middlewares {
		err := mware(acc)
		if err != nil {
			return err
		}
	}

	// fire the response to the queryer's url
	resp, err := q.client.Do(req)
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
