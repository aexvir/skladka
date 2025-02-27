package errors

import (
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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

// Format implements the fmt.Formatter interface to customize how the error is formatted.
// It supports the following format verbs:
//
//	%s    prints just the error message
//	%q    wraps the error message in quotes
//	%v    same as %s
//	%+v   prints error message, wrapped error if any, and stack trace
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
	var httperr *HTTPError

	if stderr.As(err, &httperr) {
		return httperr
	}

	return &HTTPError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Server Error",
		error:   err,
		stack:   callers(),
	}
}

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

// WithErrorHandler wraps an ErrorHandlerFunc and returns an http.HandlerFunc.
// It handles errors returned by the ErrorHandlerFunc by:
//   - For HTTPError: responding with the error's code and message
//   - For other errors: setting the OpenTelemetry span status to Error and
//     responding with 500 Internal Server Error
//
// Example usage:
//
//	http.HandleFunc(
//	    "/users", errors.WithErrorHandler(
//	        func(w http.ResponseWriter, r *http.Request) error {
//	            user, err := getUser(r.Context())
//	            if err != nil {
//	                return errors.NewHTTPError(http.StatusNotFound, "user not found", err)
//	            }
//	            return json.NewEncoder(w).Encode(user)
//	        }
//	    )
//	)
func WithErrorHandler(handler ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			var httperr *HTTPError

			if stderr.As(err, &httperr) {
				http.Error(w, httperr.Error(), httperr.Code)
				return
			}

			span := trace.SpanFromContext(r.Context())
			span.SetStatus(codes.Error, err.Error())

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
