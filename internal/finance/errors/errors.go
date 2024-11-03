package errors

import (
	"errors"
	"fmt"
	"strings"
)

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

func NewValidationError(msg string) error {
	return &ValidationError{Msg: msg}
}

func IsValidationError(err error) bool {
	var validationError *ValidationError
	ok := errors.As(err, &validationError)
	return ok
}

func NewIndexedValidationError(index int, msg string) error {
	return &ValidationError{Msg: fmt.Sprintf("Validation error at transaction %d: %s", index, msg)}
}

var ErrInvalidUserCategory = NewValidationError("Invalid personal user category")
var ErrInvalidPredefinedCategory = NewValidationError("Invalid predefined category")

type ValidationErrors struct {
	Errors []error
}

func (ve *ValidationErrors) Error() string {
	errorMessages := make([]string, len(ve.Errors))
	for i, err := range ve.Errors {
		errorMessages[i] = err.Error()
	}
	return fmt.Sprintf("multiple validation errors: %s", strings.Join(errorMessages, "; "))
}

func (ve *ValidationErrors) Add(err error) {
	ve.Errors = append(ve.Errors, err)
}

func IsValidationErrors(err error) bool {
	var validationErrors *ValidationErrors
	ok := errors.As(err, &validationErrors)
	return ok
}
