package errors

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		err  string
		want error
	}{
		{"", fmt.Errorf("")},
		{"foo", fmt.Errorf("foo")},
		{"foo", New("foo")},
		{"string with format specifiers: %v", errors.New("string with format specifiers: %v")},
	}

	for _, test := range tests {
		got := New(test.err)
		if got.Error() != test.want.Error() {
			t.Errorf("New.Error(): got: %q, want %q", got, test.want)
		}
	}
}

func TestWrapNil(t *testing.T) {
	got := Wrap(nil, "no error")
	if got != nil {
		t.Errorf("Wrap(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{Wrap(io.EOF, "read error"), "client error", "client error: read error: EOF"},
	}

	for _, test := range tests {
		got := Wrap(test.err, test.message).Error()
		if got != test.want {
			t.Errorf("Wrap(%v, %q): got: %v, want %v", test.err, test.message, got, test.want)
		}
	}
}

type nilerr struct{}

func (nilerr) Error() string { return "nil error" }

func TestCause(t *testing.T) {
	base := New("error")

	tests := []struct {
		err  error
		want error
	}{
		{
			// nil error is nil
			err:  nil,
			want: nil,
		},
		{
			// explicit nil error is nil
			err:  (error)(nil),
			want: nil,
		},
		{
			// typed nil is nil
			err:  (*nilerr)(nil),
			want: (*nilerr)(nil),
		},
		{
			// uncaused error is unaffected
			err:  io.EOF,
			want: io.EOF,
		},
		{
			// caused error returns cause
			err:  Wrap(io.EOF, "ignored"),
			want: io.EOF,
		},
		{
			err:  base, // return from errors.New
			want: base,
		},
		{
			WithMessage(nil, "whoops"),
			nil,
		},
		{
			WithMessage(io.EOF, "whoops"),
			io.EOF,
		},
		{
			WithStack(nil),
			nil,
		},
		{
			WithStack(io.EOF),
			io.EOF,
		},
	}

	for i, test := range tests {
		got := Cause(test.err)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("test %d: got %#v, want %#v", i+1, got, test.want)
		}
	}
}

func TestWrapfNil(t *testing.T) {
	got := Wrapf(nil, "no error")
	if got != nil {
		t.Errorf("Wrapf(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{Wrapf(io.EOF, "read error without format specifiers"), "client error", "client error: read error without format specifiers: EOF"},
		{Wrapf(io.EOF, "read error with %d format specifier", 1), "client error", "client error: read error with 1 format specifier: EOF"},
	}

	for _, test := range tests {
		got := Wrapf(test.err, test.message).Error()
		if got != test.want {
			t.Errorf("Wrapf(%v, %q): got: %v, want %v", test.err, test.message, got, test.want)
		}
	}
}

func TestErrorf(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{Errorf("read error without format specifiers"), "read error without format specifiers"},
		{Errorf("read error with %d format specifier", 1), "read error with 1 format specifier"},
	}

	for _, test := range tests {
		got := test.err.Error()
		if got != test.want {
			t.Errorf("Errorf(%v): got: %q, want %q", test.err, got, test.want)
		}
	}
}

func TestWithStackNil(t *testing.T) {
	got := WithStack(nil)
	if got != nil {
		t.Errorf("WithStack(nil): got %#v, expected nil", got)
	}
}

func TestWithStack(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{io.EOF, "EOF"},
		{WithStack(io.EOF), "EOF"},
	}

	for _, test := range tests {
		got := WithStack(test.err).Error()
		if got != test.want {
			t.Errorf("WithStack(%v): got: %v, want %v", test.err, got, test.want)
		}
	}
}

func TestWithMessageNil(t *testing.T) {
	got := WithMessage(nil, "no error")
	if got != nil {
		t.Errorf("WithMessage(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWithMessage(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{WithMessage(io.EOF, "read error"), "client error", "client error: read error: EOF"},
	}

	for _, test := range tests {
		got := WithMessage(test.err, test.message).Error()
		if got != test.want {
			t.Errorf("WithMessage(%v, %q): got: %q, want %q", test.err, test.message, got, test.want)
		}
	}
}

func TestWithMessagefNil(t *testing.T) {
	got := WithMessagef(nil, "no error")
	if got != nil {
		t.Errorf("WithMessage(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWithMessagef(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{WithMessagef(io.EOF, "read error without format specifier"), "client error", "client error: read error without format specifier: EOF"},
		{WithMessagef(io.EOF, "read error with %d format specifier", 1), "client error", "client error: read error with 1 format specifier: EOF"},
	}

	for _, test := range tests {
		got := WithMessagef(test.err, test.message).Error()
		if got != test.want {
			t.Errorf("WithMessage(%v, %q): got: %q, want %q", test.err, test.message, got, test.want)
		}
	}
}

// errors.New, etc values are not expected to be compared by value
// but the change in errors#27 made them incomparable. Assert that
// various kinds of errors have a functional equality operator, even
// if the result of that equality is always false.
func TestErrorEquality(t *testing.T) {
	vals := []error{
		nil,
		io.EOF,
		errors.New("EOF"),
		New("EOF"),
		Errorf("EOF"),
		Wrap(io.EOF, "EOF"),
		Wrapf(io.EOF, "EOF%d", 2),
		WithMessage(nil, "whoops"),
		WithMessage(io.EOF, "whoops"),
		WithStack(io.EOF),
		WithStack(nil),
	}

	for i := range vals {
		for j := range vals {
			_ = vals[i] == vals[j] // mustn't panic
		}
	}
}

func TestFormatNew(t *testing.T) {
	tests := []struct {
		error
		format string
		want   string
	}{
		{
			New("error"),
			"%s",
			"error",
		},
		{
			New("error"),
			"%v",
			"error",
		},
		{
			New("error"),
			"%+v",
			"error\n" +
				"errors.TestFormatNew\n" +
				"\t.+/errors/errors_test.go:284",
		},
		{
			New("error"),
			"%q",
			`"error"`,
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.error, test.format, test.want)
	}
}

func TestFormatErrorf(t *testing.T) {
	tests := []struct {
		error
		format string
		want   string
	}{
		{
			Errorf("%s", "error"),
			"%s",
			"error",
		},
		{
			Errorf("%s", "error"),
			"%v",
			"error",
		},
		{
			Errorf("%s", "error"),
			"%+v",
			"error\n" +
				"errors.TestFormatErrorf\n" +
				"\t.+/errors/errors_test.go:319",
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.error, test.format, test.want)
	}
}

func TestFormatWrap(t *testing.T) {
	tests := []struct {
		error
		format string
		want   string
	}{
		{
			Wrap(New("error"), "error2"),
			"%s",
			"error2: error",
		},
		{
			Wrap(New("error"), "error2"),
			"%v",
			"error2: error",
		},
		{
			Wrap(New("error"), "error2"),
			"%+v",
			"error\n" +
				"errors.TestFormatWrap\n" +
				"\t.+/errors/errors_test.go:349",
		},
		{
			Wrap(io.EOF, "error"),
			"%s",
			"error: EOF",
		},
		{
			Wrap(io.EOF, "error"),
			"%v",
			"error: EOF",
		},
		{
			Wrap(io.EOF, "error"),
			"%+v",
			"EOF\n" +
				"error\n" +
				"errors.TestFormatWrap\n" +
				"\t.+/errors/errors_test.go:366",
		},
		{
			Wrap(Wrap(io.EOF, "error1"), "error2"),
			"%+v",
			"EOF\n" +
				"error1\n" +
				"errors.TestFormatWrap\n" +
				"\t.+/errors/errors_test.go:374\n",
		},
		{
			Wrap(New("error with space"), "context"),
			"%q",
			`"context: error with space"`,
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.error, test.format, test.want)
	}
}

func TestFormatWrapf(t *testing.T) {
	tests := []struct {
		error
		format string
		want   string
	}{
		{
			Wrapf(io.EOF, "error%d", 2),
			"%s",
			"error2: EOF",
		},
		{
			Wrapf(io.EOF, "error%d", 2),
			"%v",
			"error2: EOF",
		},
		{
			Wrapf(io.EOF, "error%d", 2),
			"%+v",
			"EOF\n" +
				"error2\n" +
				"errors.TestFormatWrapf\n" +
				"\t.+/errors/errors_test.go:410",
		},
		{
			Wrapf(New("error"), "error%d", 2),
			"%s",
			"error2: error",
		},
		{
			Wrapf(New("error"), "error%d", 2),
			"%v",
			"error2: error",
		},
		{
			Wrapf(New("error"), "error%d", 2),
			"%+v",
			"error\n" +
				"errors.TestFormatWrapf\n" +
				"\t.+/errors/errors_test.go:428",
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.error, test.format, test.want)
	}
}

func TestFormatWithStack(t *testing.T) {
	tests := []struct {
		error
		format string
		want   []string
	}{
		{
			WithStack(io.EOF),
			"%s",
			[]string{"EOF"},
		},
		{
			WithStack(io.EOF),
			"%v",
			[]string{"EOF"},
		},
		{
			WithStack(io.EOF),
			"%+v",
			[]string{
				"EOF",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:458",
			},
		},
		{
			WithStack(New("error")),
			"%s",
			[]string{"error"},
		},
		{
			WithStack(New("error")),
			"%v",
			[]string{"error"},
		},
		{
			WithStack(New("error")),
			"%+v",
			[]string{
				"error",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:477",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:477",
			},
		},
		{
			WithStack(WithStack(io.EOF)),
			"%+v",
			[]string{
				"EOF",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:488",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:488",
			},
		},
		{
			WithStack(WithStack(Wrapf(io.EOF, "message"))),
			"%+v",
			[]string{
				"EOF",
				"message",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:499",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:499",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:499",
			},
		},
		{
			WithStack(Errorf("error%d", 1)),
			"%+v",
			[]string{
				"error1",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:513",
				"errors.TestFormatWithStack\n" +
					"\t.+/errors/errors_test.go:513",
			},
		},
	}

	for i, test := range tests {
		testFormatCompleteCompare(t, i, test.error, test.format, test.want, true)
	}
}

func TestFormatWithMessage(t *testing.T) {
	tests := []struct {
		error
		format string
		want   []string
	}{
		{
			WithMessage(New("error"), "error2"),
			"%s",
			[]string{"error2: error"},
		},
		{
			WithMessage(New("error"), "error2"),
			"%v",
			[]string{"error2: error"},
		},
		{
			WithMessage(New("error"), "error2"),
			"%+v",
			[]string{
				"error",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:547",
				"error2",
			},
		},
		{
			WithMessage(io.EOF, "addition1"),
			"%s",
			[]string{"addition1: EOF"},
		},
		{
			WithMessage(io.EOF, "addition1"),
			"%v",
			[]string{"addition1: EOF"},
		},
		{
			WithMessage(io.EOF, "addition1"),
			"%+v",
			[]string{"EOF", "addition1"},
		},
		{
			WithMessage(WithMessage(io.EOF, "addition1"), "addition2"),
			"%v",
			[]string{"addition2: addition1: EOF"},
		},
		{
			WithMessage(WithMessage(io.EOF, "addition1"), "addition2"),
			"%+v",
			[]string{"EOF", "addition1", "addition2"},
		},
		{
			Wrap(WithMessage(io.EOF, "error1"), "error2"),
			"%+v",
			[]string{
				"EOF", "error1", "error2",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:582",
			},
		},
		{
			WithMessage(Errorf("error%d", 1), "error2"),
			"%+v",
			[]string{
				"error1",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:591",
				"error2",
			},
		},
		{
			WithMessage(WithStack(io.EOF), "error"),
			"%+v",
			[]string{
				"EOF",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:601",
				"error",
			},
		},
		{
			WithMessage(Wrap(WithStack(io.EOF), "inside-error"), "outside-error"),
			"%+v",
			[]string{
				"EOF",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:611",
				"inside-error",
				"errors.TestFormatWithMessage\n" +
					"\t.+/errors/errors_test.go:611",
				"outside-error",
			},
		},
	}

	for i, test := range tests {
		testFormatCompleteCompare(t, i, test.error, test.format, test.want, true)
	}
}

func TestFormatGeneric(t *testing.T) {
	starts := []struct {
		err  error
		want []string
	}{
		{
			New("new-error"),
			[]string{
				"new-error",
				"errors.TestFormatGeneric\n" +
					"\t.+/errors/errors_test.go:636",
			},
		},
		{
			Errorf("errorf-error"),
			[]string{
				"errorf-error",
				"errors.TestFormatGeneric\n" +
					"\t.+/errors/errors_test.go:644",
			},
		},
		{
			errors.New("errors-new-error"),
			[]string{
				"errors-new-error",
			},
		},
	}

	wrappers := []wrapper{
		{
			func(err error) error { return WithMessage(err, "with-message") },
			[]string{"with-message"},
		},
		{
			func(err error) error { return WithStack(err) },
			[]string{
				"errors.(func·002|TestFormatGeneric.func2)\n\t" +
					".+/errors/errors_test.go:665",
			},
		},
		{
			func(err error) error { return Wrap(err, "wrap-error") },
			[]string{
				"wrap-error",
				"errors.(func·003|TestFormatGeneric.func3)\n\t" +
					".+/errors/errors_test.go:672",
			},
		},
		{
			func(err error) error { return Wrapf(err, "wrapf-error%d", 1) },
			[]string{
				"wrapf-error1",
				"errors.(func·004|TestFormatGeneric.func4)\n\t" +
					".+/errors/errors_test.go:680",
			},
		},
	}

	for s := range starts {
		err := starts[s].err
		want := starts[s].want
		testFormatCompleteCompare(t, s, err, "%+v", want, false)
		testGenericRecursive(t, err, want, wrappers, 3)
	}
}

func wrappedNew(message string) error { // This function will be mid-stack inlined in go 1.12+
	return New(message)
}

func TestFormatWrappedNew(t *testing.T) {
	tests := []struct {
		error
		format string
		want   string
	}{
		{
			wrappedNew("error"),
			"%+v",
			"error\n" +
				"errors.wrappedNew\n" +
				"\t.+/errors/errors_test.go:698\n" +
				"errors.TestFormatWrappedNew\n" +
				"\t.+/errors/errors_test.go:708",
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.error, test.format, test.want)
	}
}

func TestJoin(t *testing.T) {
	err1 := New("error 1")
	err2 := New("error 2")

	tests := []struct {
		name     string
		errs     []error
		wantNil  bool
		wantText string
	}{
		{
			name:     "no errors",
			errs:     []error{},
			wantNil:  true,
			wantText: "",
		},
		{
			name:     "nil errors",
			errs:     []error{nil, nil},
			wantNil:  true,
			wantText: "",
		},
		{
			name:     "single error",
			errs:     []error{err1},
			wantText: "error 1",
		},
		{
			name:     "multiple errors",
			errs:     []error{err1, err2},
			wantText: "error 1\nerror 2",
		},
		{
			name:     "with nil errors",
			errs:     []error{nil, err1, nil, err2, nil},
			wantText: "error 1\nerror 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Join(tt.errs...)
			if tt.wantNil {
				if err != nil {
					t.Errorf("Join() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Join() = nil, want error")
			}
			if got := err.Error(); got != tt.wantText {
				t.Errorf("Join().Error() = %q, want %q", got, tt.wantText)
			}

			// Test unwrap
			if jerr, ok := err.(*withMultierr); !ok {
				t.Error("Join() did not return a *joinError")
			} else {
				errs := jerr.Unwrap()
				var nonNilErrs []error
				for _, err := range tt.errs {
					if err != nil {
						nonNilErrs = append(nonNilErrs, err)
					}
				}
				if len(errs) != len(nonNilErrs) {
					t.Errorf("Join().Unwrap() returned %d errors, want %d", len(errs), len(nonNilErrs))
				}
			}

			// Test stack trace
			if st, ok := err.(interface{ StackTrace() StackTrace }); !ok {
				t.Error("Join() error does not implement StackTrace")
			} else if len(st.StackTrace()) == 0 {
				t.Error("Join() error has empty stack trace")
			}
		})
	}
}

func TestJoinFormat(t *testing.T) {
	err1 := New("error 1")
	err2 := Wrap(New("inner"), "error 2")

	err := Join(err1, err2)

	formatted := fmt.Sprintf("%+v", err)
	if !strings.Contains(formatted, "error 1") {
		t.Error("formatted error does not contain first error")
	}
	if !strings.Contains(formatted, "error 2") {
		t.Error("formatted error does not contain second error")
	}
	if !strings.Contains(formatted, "inner") {
		t.Error("formatted error does not contain wrapped error")
	}
	if !strings.Contains(formatted, "TestJoinFormat") {
		t.Error("formatted error does not contain stack trace")
	}
}

func testFormatRegexp(t *testing.T, n int, arg interface{}, format, want string) {
	t.Helper()
	got := fmt.Sprintf(format, arg)

	gotLines := strings.SplitN(got, "\n", -1)
	wantLines := strings.SplitN(want, "\n", -1)

	if len(wantLines) > len(gotLines) {
		t.Errorf("test %d: wantLines(%d) > gotLines(%d):\n got: %q\nwant: %q", n+1, len(wantLines), len(gotLines), got, want)
		return
	}

	for idx, wantLine := range wantLines {
		match, err := regexp.MatchString(wantLine, gotLines[idx])
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			t.Errorf("test %d: line %d: fmt.Sprintf(%q, err):\n got: %q\nwant: %q", n+1, idx+1, format, got, want)
		}
	}
}

var stackLineR = regexp.MustCompile(`\.`)

// parseBlocks parses input into a slice, where:
//   - incase entry contains a newline, its a stacktrace
//   - incase entry contains no newline, its a solo line.
//
// Detecting stack boundaries only works incase the WithStack-calls are
// to be found on the same line, thats why it is optionally here.
//
// Example use:
//
//	for _, e := range blocks {
//	  if strings.ContainsAny(e, "\n") {
//	    // Match as stack
//	  } else {
//	    // Match as line
//	  }
//	}
func parseBlocks(input string, detectStackboundaries bool) ([]string, error) {
	var blocks []string

	stack := ""
	wasStack := false
	lines := map[string]bool{} // already found lines

	for _, l := range strings.Split(input, "\n") {
		isStackLine := stackLineR.MatchString(l)

		switch {
		case !isStackLine && wasStack:
			blocks = append(blocks, stack, l)
			stack = ""
			lines = map[string]bool{}
		case isStackLine:
			if wasStack {
				// Detecting two stacks after another, possible cause lines match in
				// our tests due to WithStack(WithStack(io.EOF)) on same line.
				if detectStackboundaries {
					if lines[l] {
						if len(stack) == 0 {
							return nil, errors.New("len of block must not be zero here")
						}

						blocks = append(blocks, stack)
						stack = l
						lines = map[string]bool{l: true}
						continue
					}
				}

				stack = stack + "\n" + l
			} else {
				stack = l
			}
			lines[l] = true
		case !isStackLine && !wasStack:
			blocks = append(blocks, l)
		default:
			return nil, errors.New("must not happen")
		}

		wasStack = isStackLine
	}

	// Use up stack
	if stack != "" {
		blocks = append(blocks, stack)
	}
	return blocks, nil
}

func testFormatCompleteCompare(t *testing.T, n int, arg interface{}, format string, want []string, detectStackBoundaries bool) {
	gotStr := fmt.Sprintf(format, arg)

	got, err := parseBlocks(gotStr, detectStackBoundaries)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != len(want) {
		t.Fatalf("test %d: fmt.Sprintf(%s, err) -> wrong number of blocks: got(%d) want(%d)\n got: %s\nwant: %s\ngotStr: %q",
			n+1, format, len(got), len(want), prettyBlocks(got), prettyBlocks(want), gotStr)
	}

	for i := range got {
		if strings.ContainsAny(want[i], "\n") {
			// Match as stack
			match, err := regexp.MatchString(want[i], got[i])
			if err != nil {
				t.Fatal(err)
			}
			if !match {
				t.Fatalf("test %d: block %d: fmt.Sprintf(%q, err):\ngot:\n%q\nwant:\n%q\nall-got:\n%s\nall-want:\n%s\n",
					n+1, i+1, format, got[i], want[i], prettyBlocks(got), prettyBlocks(want))
			}
		} else {
			// Match as message
			if got[i] != want[i] {
				t.Fatalf("test %d: fmt.Sprintf(%s, err) at block %d got != want:\n got: %q\nwant: %q", n+1, format, i+1, got[i], want[i])
			}
		}
	}
}

type wrapper struct {
	wrap func(err error) error
	want []string
}

func prettyBlocks(blocks []string) string {
	var out []string

	for _, b := range blocks {
		out = append(out, fmt.Sprintf("%v", b))
	}

	return "   " + strings.Join(out, "\n   ")
}

func testGenericRecursive(t *testing.T, beforeErr error, beforeWant []string, list []wrapper, maxDepth int) {
	if len(beforeWant) == 0 {
		panic("beforeWant must not be empty")
	}
	for _, w := range list {
		if len(w.want) == 0 {
			panic("want must not be empty")
		}

		err := w.wrap(beforeErr)

		// Copy required cause append(beforeWant, ..) modified beforeWant subtly.
		beforeCopy := make([]string, len(beforeWant))
		copy(beforeCopy, beforeWant)

		beforeWant := beforeCopy
		last := len(beforeWant) - 1
		var want []string

		// Merge two stacks behind each other.
		if strings.ContainsAny(beforeWant[last], "\n") && strings.ContainsAny(w.want[0], "\n") {
			want = append(beforeWant[:last], append([]string{beforeWant[last] + "((?s).*)" + w.want[0]}, w.want[1:]...)...)
		} else {
			want = append(beforeWant, w.want...)
		}

		testFormatCompleteCompare(t, maxDepth, err, "%+v", want, false)
		if maxDepth > 0 {
			testGenericRecursive(t, err, want, list, maxDepth-1)
		}
	}
}
