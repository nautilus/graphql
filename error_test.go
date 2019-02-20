package graphql

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeError(t *testing.T) {
	// marshal the 2 kinds of errors
	errWithCode, _ := json.Marshal(NewError("ERROR_CODE", "foo"))
	expected, _ := json.Marshal(map[string]interface{}{
		"extensions": map[string]interface{}{
			"code": "ERROR_CODE",
		},
		"message": "foo",
	})

	assert.Equal(t, string(expected), string(errWithCode))
}
