package errors

import (
	stderr "errors"
	"fmt"
	"io"
	"strings"
)

// New returns a new error with a stack trace.
// The error implements fmt.Formatter for custom error formatting.
//
//	err := errors.New("connection failed")
//	fmt.Printf("%+v", err) // prints error with stack trace
func New(message string) error {
	return &fundamental{
		msg:   message,
		stack: callers(),
	}
}

// Errorf returns a new error with formatted message and a stack trace.
// The error implements fmt.Formatter for custom error formatting.
//
//	name := "db"
//	err := errors.Errorf("connection to %s failed", name)
func Errorf(format string, args ...interface{}) error {
	return &fundamental{
		msg:   fmt.Sprintf(format, args...),
		stack: callers(),
	}
}

// Wrap returns an error annotating err with a stack trace at the point Wrap is called,
// and the supplied message. If err is nil, Wrap returns nil.
//
//	err := doSomething()
//	if err != nil {
//	    return errors.Wrap(err, "failed to do something")
//	}
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	return &withStack{
		&withMessage{
			cause: err,
			msg:   message,
		},
		callers(),
	}
}

// Wrapf returns an error annotating err with a stack trace at the point Wrapf is called,
// and the format specifier. If err is nil, Wrapf returns nil.
//
//	err := doSomething()
//	if err != nil {
//	    return errors.Wrapf(err, "failed to do something: %s", details)
//	}
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	return &withStack{
		&withMessage{
			cause: err,
			msg:   fmt.Sprintf(format, args...),
		},
		callers(),
	}
}

// WithStack annotates err with a stack trace at the point WithStack was called.
// If err is nil, WithStack returns nil.
//
//	if err != nil {
//	    return errors.WithStack(err)
//	}
func WithStack(err error) error {
	if err == nil {
		return nil
	}
	return &withStack{
		err,
		callers(),
	}
}

// WithMessage annotates err with a new message.
// If err is nil, WithMessage returns nil.
//
//	err := doSomething()
//	if err != nil {
//	    return errors.WithMessage(err, "failed to do something")
//	}
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   message,
		stack: callers(),
	}
}

// WithMessagef annotates err with the format specifier.
// If err is nil, WithMessagef returns nil.
//
//	err := doSomething()
//	if err != nil {
//	    return errors.WithMessagef(err, "failed to do something: %s", details)
//	}
func WithMessagef(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		cause: err,
		msg:   fmt.Sprintf(format, args...),
		stack: callers(),
	}
}

// Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following interface:
//
//	type causer interface {
//	    Cause() error
//	}
//
// If the error does not implement Cause, the original error will be returned.
// If the error is nil, nil will be returned without further investigation.
func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}

	return err
}

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// Join returns nil if errs contains no non-nil values.
// The error formats as the concatenation of the strings obtained
// by calling the Error method of each element of errs, with a newline
// between each string.
//
//	err1 := errors.New("first error")
//	err2 := errors.New("second error")
//	err := errors.Join(err1, err2)
//	fmt.Printf("%v", err)
//	// Output:
//	// first error
//	// second error
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &withMultierr{
		errs:  make([]error, 0, n),
		stack: callers(),
	}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}

// As finds the first error in err's chain that matches target,
// and if one is found, As returns true and sets target to that error value.
// Otherwise, As returns false and sets target to nil.
func As(err error, target any) bool {
	return stderr.As(err, target)
}

// Is reports whether any error in errs matches target.
// An error matches target if the error values are identical, as determined by ==.
func Is(err, target error) bool {
	return stderr.Is(err, target)
}

// fundamental represents a base error with a message and a stack trace.
// It implements error, fmt.Formatter, and errors.Is interfaces.
type fundamental struct {
	msg string
	*stack
}

func (f *fundamental) Error() string { return f.msg }

func (f *fundamental) Is(target error) bool {
	t, ok := target.(*fundamental)
	if !ok {
		return false
	}
	return f.msg == t.msg
}

func (f *fundamental) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.msg)
			f.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.msg)
	case 'q':
		fmt.Fprintf(s, "%q", f.msg)
	}
}

// withStack annotates an error with a stack trace at the point withStack was called.
// If the error does not have a stack trace, withStack provides one.
type withStack struct {
	error
	*stack
}

func (w *withStack) Cause() error { return w.error }

// Unwrap implements the errors.Unwrap interface.
// This allows the error to work with errors.Is, errors.As and errors.Unwrap.
func (w *withStack) Unwrap() error { return w.error }

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v", w.Cause())
			w.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func (w *withStack) Is(target error) bool {
	return stderr.Is(w.error, target)
}

// withMessage annotates an error with a message.
// The underlying error can be retrieved with errors.Unwrap.
type withMessage struct {
	cause error
	msg   string
	*stack
}

func (w *withMessage) Error() string { return w.msg + ": " + w.cause.Error() }
func (w *withMessage) Cause() error  { return w.cause }

// Unwrap implements the errors.Unwrap interface.
// This allows the error to work with errors.Is, errors.As and errors.Unwrap.
func (w *withMessage) Unwrap() error { return w.cause }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", w.Cause())
			io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	default:
		io.WriteString(s, w.Error())
	}
}

func (w *withMessage) Is(target error) bool {
	return stderr.Is(w.cause, target)
}

// withMultierr combines multiple errors into a single error value.
// It implements error.Unwrap to return the list of underlying errors.
type withMultierr struct {
	errs []error
	*stack
}

func (j *withMultierr) Error() string {
	var b []string
	for _, err := range j.errs {
		b = append(b, err.Error())
	}
	return strings.Join(b, "\n")
}

// Unwrap implements the errors.Unwrap interface for multiple errors.
// This allows the error to work with errors.Is and errors.As across
// all contained errors.
func (j *withMultierr) Unwrap() []error { return j.errs }

func (j *withMultierr) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			for i, err := range j.errs {
				if i > 0 {
					io.WriteString(s, "\n")
				}
				fmt.Fprintf(s, "%+v", err)
			}
			j.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, j.Error())
	case 'q':
		fmt.Fprintf(s, "%q", j.Error())
	}
}

func (j *withMultierr) Is(target error) bool {
	for _, err := range j.errs {
		if stderr.Is(err, target) {
			return true
		}
	}
	return false
}
