package graphql

import "strings"

// ErrorExtensions define fields that extend the standard graphql error shape
type ErrorExtensions struct {
	Code 		string `json:"code"`
}

// Error represents a graphql error
type Error struct {
	Extensions ErrorExtensions 	`json:"extensions"`
	Message    string          	`json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

// NewError returns a graphql error with the given code and message
func NewError(code string, message string) *Error {
	return &Error{
		Message: message,
		Extensions: ErrorExtensions{
			Code: code,
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
