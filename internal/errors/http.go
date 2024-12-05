package errors

import (
	"fmt"
	"io"
	"net/http"

	stderr "errors"
)

// HTTPError represents an error that includes an HTTP status code.
// It implements error, fmt.Formatter, and errors.Unwrap interfaces.
// The error can be formatted with %s, %q, %v, and %+v verbs.
//
// Example usage:
//
//	err := errors.NewHTTPError(http.StatusNotFound, "user not found", nil)
//	fmt.Printf("%+v\n", err) // prints detailed error with stack trace
//	fmt.Printf("%v\n", err)  // prints "user not found"
//	fmt.Printf("%q\n", err)  // prints "\"user not found\""
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	error   error
	*stack
}

func (e *HTTPError) Error() string {
	if e.error != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.error)
	}
	return e.Message
}

// Unwrap implements the errors.Unwrap interface.
// This allows the error to work with errors.Is, errors.As and errors.Unwrap.
func (e *HTTPError) Unwrap() error {
	return e.error
}

func (e *HTTPError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%s\n", e.Message)
			if e.error != nil {
				fmt.Fprintf(s, "%+v", e.error)
			}
			e.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}

// Is reports whether this error matches target.
// An error matches if both the Code and Message are equal.
func (e *HTTPError) Is(target error) bool {
	t, ok := target.(*HTTPError)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Message == t.Message
}

// NewHTTPError creates a new HTTPError with the given code and message.
// If err is not nil, it will be wrapped and accessible via Unwrap().
//
//	err := errors.NewHTTPError(http.StatusNotFound, "user not found", nil)
//	err = errors.NewHTTPError(http.StatusInternalServerError, "database error", dbErr)
func NewHTTPError(code int, message string, err error) error {
	return &HTTPError{
		Code:    code,
		Message: message,
		error:   err,
		stack:   callers(),
	}
}

// AsHTTPError attempts to convert an error to an HTTPError.
// If the error is already an HTTPError, it is returned as is.
// If the error is not an HTTPError, it is wrapped as an internal server error.
//
//	err := someFunction()
//	httpErr := errors.AsHTTPError(err)
//	fmt.Printf("Status code: %d\n", httpErr.Code)
func AsHTTPError(err error) *HTTPError {
	var httpErr *HTTPError
	if stderr.As(err, &httpErr) {
		return httpErr
	}
	return &HTTPError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Server Error",
		error:   err,
		stack:   callers(),
	}
}

var (
	// Common application errors
	ErrNotFound           = NewHTTPError(http.StatusNotFound, "Resource not found", nil)
	ErrBadRequest         = NewHTTPError(http.StatusBadRequest, "Bad request", nil)
	ErrUnauthorized       = NewHTTPError(http.StatusUnauthorized, "Unauthorized", nil)
	ErrForbidden          = NewHTTPError(http.StatusForbidden, "Forbidden", nil)
	ErrInternalServer     = NewHTTPError(http.StatusInternalServerError, "Internal server error", nil)
	ErrServiceUnavailable = NewHTTPError(http.StatusServiceUnavailable, "Service unavailable", nil)
)

// IsNotFound returns true if the error is a not found error
func IsNotFound(err error) bool {
	var httpErr *HTTPError
	return stderr.As(err, &httpErr) && httpErr.Code == http.StatusNotFound
}

// IsBadRequest returns true if the error is a bad request error
func IsBadRequest(err error) bool {
	var httpErr *HTTPError
	return stderr.As(err, &httpErr) && httpErr.Code == http.StatusBadRequest
}

// IsUnauthorized returns true if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	var httpErr *HTTPError
	return stderr.As(err, &httpErr) && httpErr.Code == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a forbidden error
func IsForbidden(err error) bool {
	var httpErr *HTTPError
	return stderr.As(err, &httpErr) && httpErr.Code == http.StatusForbidden
}

// IsInternalServer returns true if the error is an internal server error
func IsInternalServer(err error) bool {
	var httpErr *HTTPError
	return stderr.As(err, &httpErr) && httpErr.Code == http.StatusInternalServerError
}
