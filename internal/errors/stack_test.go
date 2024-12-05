package errors

import (
	"fmt"
	"runtime"
	"testing"
)

var initpc = caller()

type X struct{}

// val returns a Frame pointing to itself.
func (x X) val() Frame {
	return caller()
}

// ptr returns a Frame pointing to itself.
func (x *X) ptr() Frame {
	return caller()
}

func TestFrameFormat(t *testing.T) {
	tests := []struct {
		Frame
		format string
		want   string
	}{
		{
			initpc,
			"%s",
			"stack_test.go",
		},
		{
			initpc,
			"%+s",
			".+/errors.init\n" +
				"\t.+/errors/stack_test.go",
		},
		{
			0,
			"%s",
			"unknown",
		},
		{
			0,
			"%+s",
			"unknown",
		},
		{
			initpc,
			"%d",
			"9",
		},
		{
			0,
			"%d",
			"0",
		},
		{
			initpc,
			"%n",
			"init",
		},
		{
			func() Frame {
				var x X
				return x.ptr()
			}(),
			"%n",
			`\(\*X\).ptr`,
		},
		{
			func() Frame {
				var x X
				return x.val()
			}(),
			"%n",
			"X.val",
		},
		{
			0,
			"%n",
			"",
		},
		{
			initpc,
			"%v",
			"stack_test.go:9",
		},
		{
			initpc,
			"%+v",
			"errors.init\n" +
				"\t.+/errors/stack_test.go:9",
		},
		{
			0,
			"%v",
			"unknown:0",
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.Frame, test.format, test.want)
	}
}

func TestFuncname(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"", ""},
		{"runtime.main", "main"},
		{"errors.funcname", "funcname"},
		{"funcname", "funcname"},
		{"io.copyBuffer", "copyBuffer"},
		{"main.(*R).Write", "(*R).Write"},
	}

	for _, test := range tests {
		got := funcname(test.name)
		want := test.want
		if got != want {
			t.Errorf("funcname(%q): want: %q, got %q", test.name, want, got)
		}
	}
}

func TestStackTrace(t *testing.T) {
	tests := []struct {
		err  error
		want []string
	}{
		{
			New("ooh"),
			[]string{
				".+errors.TestStackTrace\n" +
					"\t.+/errors/stack_test.go:136",
			},
		},
		{
			Wrap(
				New("ooh"),
				"ahh",
			),
			[]string{
				".+errors.TestStackTrace\n" +
					"\t.+/errors/stack_test.go:143", // this is the stack of Wrap, not New
			},
		},
		{
			Cause(
				Wrap(
					New("ooh"),
					"ahh",
				),
			),
			[]string{
				".+errors.TestStackTrace\n" +
					"\t.+/errors/stack_test.go:155", // this is the stack of New
			},
		},
		{
			func() error {
				return New("ooh")
			}(),
			[]string{
				`.+errors.TestStackTrace.func1` +
					"\n\t.+/errors/stack_test.go:166", // this is the stack of New
				".+errors.TestStackTrace\n" +
					"\t.+/errors/stack_test.go:167", // this is the stack of New's caller
			},
		},
		{
			Cause(
				func() error {
					return func() error {
						return Errorf(
							"hello %s",
							fmt.Sprintf("world: %s", "ooh"),
						)
					}()
				}(),
			),
			[]string{
				`.+errors.TestStackTrace.TestStackTrace.func2.func3` + // no idea why TestStackTrace is twice
					"\n\t.+/errors/stack_test.go:179", // this is the stack of Errorf
				`.+errors.TestStackTrace.func2` +
					"\n\t.+/errors/stack_test.go:183", // this is the stack of Errorf's caller
				".+errors.TestStackTrace\n" +
					"\t.+/errors/stack_test.go:184", // this is the stack of Errorf's caller's caller
			},
		},
	}

	type tracer interface {
		StackTrace() StackTrace
	}

	for i, test := range tests {
		err, ok := test.err.(tracer)
		if !ok {
			t.Errorf("expected %#v to implement StackTrace() StackTrace", test.err)
			continue
		}

		st := err.StackTrace()

		for j, want := range test.want {
			testFormatRegexp(t, i, st[j], "%+v", want)
		}
	}
}

func stackTrace() StackTrace {
	const depth = 8
	var pcs [depth]uintptr
	n := runtime.Callers(1, pcs[:])
	var st stack = pcs[0:n]
	return st.StackTrace()
}

func TestStackTraceFormat(t *testing.T) {
	tests := []struct {
		StackTrace
		format string
		want   string
	}{
		{
			nil,
			"%s",
			`\[\]`,
		},
		{
			nil,
			"%v",
			`\[\]`,
		},
		{
			nil,
			"%+v",
			"",
		},
		{
			nil,
			"%#v",
			`\[\]errors.Frame\(nil\)`,
		},
		{
			make(StackTrace, 0),
			"%s",
			`\[\]`,
		},
		{
			make(StackTrace, 0),
			"%v",
			`\[\]`,
		},
		{
			make(StackTrace, 0),
			"%+v",
			"",
		},
		{
			make(StackTrace, 0),
			"%#v",
			`\[\]errors.Frame{}`,
		},
		{
			stackTrace()[:2],
			"%s",
			`\[stack_test.go stack_test.go\]`,
		},
		{
			stackTrace()[:2],
			"%v",
			`\[stack_test.go:219 stack_test.go:276\]`,
		},
		{
			stackTrace()[:2],
			"%+v",
			"\n" +
				".+errors.stackTrace\n" +
				"\t.+/errors/stack_test.go:219\n" +
				".+errors.TestStackTraceFormat\n" +
				"\t.+/errors/stack_test.go:281",
		},
		{
			stackTrace()[:2],
			"%#v",
			`\[\]errors.Frame{stack_test.go:219, stack_test.go:290}`,
		},
	}

	for i, test := range tests {
		testFormatRegexp(t, i, test.StackTrace, test.format, test.want)
	}
}

// a version of runtime.Caller that returns a Frame, not a uintptr.
func caller() Frame {
	var pcs [3]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()
	return Frame(frame.PC)
}
