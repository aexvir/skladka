// Package errors provides enhanced error handling primitives that extend Go's standard
// errors package. It offers stack traces, error wrapping, and HTTP error handling
// while maintaining compatibility with the standard library.
//
// This is a copy of github.com/pkg/errors slightly updated and extended to support more
// modern logging approaches.
//
// Key Features:
//   - Stack Traces: Automatically capture stack traces when errors are created
//   - Error Wrapping: Add context to errors while preserving the original error
//   - HTTP Errors: Structured error types for HTTP responses
//   - Error Joining: Combine multiple errors into a single error value
//
// Basic Usage:
//
//	// Create a new error with stack trace
//	err := errors.New("database connection failed")
//
//	// Add context to an existing error
//	if err != nil {
//	    return errors.Wrap(err, "failed to initialize storage")
//	}
//
//	// Format error with stack trace
//	fmt.Printf("%+v\n", err)
//
// Error Types:
//
//	fundamental - Base error type with stack trace
//	withStack  - Adds stack trace to an existing error
//	withMessage - Adds a message to an existing error
//	withMultierr - Combines multiple errors into one
//	HTTPError - HTTP-specific error with status code
//
// Error Wrapping:
//
// The package provides several ways to wrap errors:
//
//	errors.Wrap(err, "message")     // Add message and stack trace
//	errors.Wrapf(err, "fmt", args)  // Add formatted message and stack trace
//	errors.WithStack(err)           // Add only stack trace
//	errors.WithMessage(err, "msg")  // Add only message
//
// Error Inspection:
//
// Use these methods to inspect errors:
//
//	errors.Is(err, target)     // Check if err matches target
//	errors.As(err, &target)    // Try to convert err to target type
//	errors.Cause(err)          // Get the root cause of the error
//
// HTTP Error Handling:
//
// For HTTP services, use the HTTP error types:
//
//	errors.NewHTTPError(http.StatusNotFound, "user not found", err)
//	errors.AsHTTPError(err)  // Convert any error to an HTTPError
//
// Formatting:
//
// All error types implement fmt.Formatter and support:
//
//	%s    - Print the error message
//	%v    - Same as %s
//	%q    - Print the error message with quotes
//	%+v   - Print detailed error with stack trace
//
// Error Joining:
//
// Combine multiple errors into one:
//
//	err1 := errors.New("first error")
//	err2 := errors.New("second error")
//	combined := errors.Join(err1, err2)
//
// Best Practices:
//
//  1. Always use errors.New() instead of errors.New() for new errors
//  2. Use errors.Wrap() when adding context to returned errors
//  3. Use %+v when logging errors to include stack traces
//  4. Use errors.Is() and errors.As() for error inspection
//  5. Use HTTPError types for HTTP API responses
package errors
