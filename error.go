package graphql

import "strings"

// Error represents a graphql error
type Error struct {
	Extensions map[string]interface{} `json:"extensions"`
	Message    string                 `json:"message"`
	Path       []interface{}          `json:"path,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// NewError returns a graphql error with the given code and message
func NewError(code string, message string) *Error {
	return &Error{
		Message: message,
		Extensions: map[string]interface{}{
			"code": code,
		},
	}
}

// ErrorList represents a list of errors
type ErrorList []error

// Error returns a string representation of each error
func (list ErrorList) Error() string {
	acc := []string{}

	for _, error := range list {
		acc = append(acc, error.Error())
	}

	return strings.Join(acc, ". ")
}
