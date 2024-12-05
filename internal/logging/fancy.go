package logging

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode"
)

// ANSI modes
const (
	ansiReset             = "\033[0m"
	ansiFaint             = "\033[2m"
	ansiResetFaint        = "\033[22m"
	ansiBrightRed         = "\033[91m"
	ansiBrightGreen       = "\033[92m"
	ansiBrightYellow      = "\033[93m"
	ansiBrightRedFaint    = "\033[91;2m"
	ansiBrightGreenFaint  = "\033[92;2m"
	ansiBrightYellowFaint = "\033[93;2m"
)

const errKey = "err"

var (
	defaultLevel      = slog.LevelInfo
	defaultTimeFormat = time.StampMilli
)

// FancyLoggerOptions for a slog.Handler that writes tinted logs. A zero FancyLoggerOptions consists
// entirely of default values.
//
// FancyLoggerOptions can be used as a drop-in replacement for [slog.HandlerOptions].
type FancyLoggerOptions struct {
	// Enable source code location (Default: false)
	AddSource bool

	// Minimum level to log (Default: slog.LevelInfo)
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See https://pkg.go.dev/log/slog#HandlerOptions for details.
	ReplaceAttr func(groups []string, attr slog.Attr) slog.Attr

	// Time format (Default: time.StampMilli)
	TimeFormat string

	// Disable color (Default: false)
	NoColor bool
}

// NewHandler creates a [slog.Handler] that writes tinted logs to Writer w,
// using the default options. If opts is nil, the default options are used.
func NewFancyHandler(w io.Writer, opts *FancyLoggerOptions) slog.Handler {
	h := &fancyhandler{
		w:          w,
		level:      defaultLevel,
		timeFormat: defaultTimeFormat,
	}
	if opts == nil {
		return h
	}

	h.addSource = opts.AddSource
	if opts.Level != nil {
		h.level = opts.Level
	}
	h.replaceAttr = opts.ReplaceAttr
	if opts.TimeFormat != "" {
		h.timeFormat = opts.TimeFormat
	}
	h.noColor = opts.NoColor
	return h
}

// fancyhandler implements a [slog.Handler].
type fancyhandler struct {
	attrsPrefix string
	groupPrefix string
	groups      []string

	mu sync.Mutex
	w  io.Writer

	addSource   bool
	level       slog.Leveler
	replaceAttr func([]string, slog.Attr) slog.Attr
	timeFormat  string
	noColor     bool
}

func (h *fancyhandler) clone() *fancyhandler {
	return &fancyhandler{
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		w:           h.w,
		addSource:   h.addSource,
		level:       h.level,
		replaceAttr: h.replaceAttr,
		timeFormat:  h.timeFormat,
		noColor:     h.noColor,
	}
}

func (h *fancyhandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *fancyhandler) Handle(_ context.Context, r slog.Record) error {
	// get a buffer from the sync pool
	buf := newBuffer()
	defer buf.Free()

	rep := h.replaceAttr

	// write time
	if !r.Time.IsZero() {
		val := r.Time.Round(0) // strip monotonic to match Attr behavior
		if rep == nil {
			h.appendTime(buf, r.Time)
			buf.WriteByte(' ')
		} else if a := rep(nil /* groups */, slog.Time(slog.TimeKey, val)); a.Key != "" {
			if a.Value.Kind() == slog.KindTime {
				h.appendTime(buf, a.Value.Time())
			} else {
				h.appendValue(buf, a.Value, false)
			}
			buf.WriteByte(' ')
		}
	}

	// write level
	if rep == nil {
		h.appendLevel(buf, r.Level)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.Any(slog.LevelKey, r.Level)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// write source
	if h.addSource {
		counters := make([]uintptr, 1024)
		n := runtime.Callers(5, counters)
		fs := runtime.CallersFrames(counters[:n])
		f, _ := fs.Next()
		if f.File != "" {
			src := &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}

			if rep == nil {
				h.appendSource(buf, src)
				buf.WriteByte(' ')
			} else if a := rep(nil /* groups */, slog.Any(slog.SourceKey, src)); a.Key != "" {
				h.appendValue(buf, a.Value, false)
				buf.WriteByte(' ')
			}
		}
	}

	// write message
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else if a := rep(nil /* groups */, slog.String(slog.MessageKey, r.Message)); a.Key != "" {
		h.appendValue(buf, a.Value, false)
		buf.WriteByte(' ')
	}

	// write handler attributes
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}

	// ensure stacktrace is always printed last
	// because it's printed with new lines
	var stack *slog.Attr

	// write attributes
	r.Attrs(
		func(attr slog.Attr) bool {
			// save stack if found, but do not print it yet
			if attr.Key == "stacktrace" || attr.Key == "error.stack" {
				stack = &attr
				return true
			}
			h.appendAttr(buf, attr, h.groupPrefix, h.groups)
			return true
		},
	)

	// if there was a stack trace, print it
	if stack != nil {
		h.appendAttr(buf, *stack, h.groupPrefix, h.groups)
	}

	if len(*buf) == 0 {
		return nil
	}
	(*buf)[len(*buf)-1] = '\n' // replace last space with newline

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := h.w.Write(*buf)
	return err
}

func (h *fancyhandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	buf := newBuffer()
	defer buf.Free()

	// write attributes to buffer
	for _, attr := range attrs {
		h.appendAttr(buf, attr, h.groupPrefix, h.groups)
	}
	h2.attrsPrefix = h.attrsPrefix + string(*buf)
	return h2
}

func (h *fancyhandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *fancyhandler) appendTime(buf *buffer, t time.Time) {
	buf.WriteStringIf(!h.noColor, ansiFaint)
	*buf = t.AppendFormat(*buf, h.timeFormat)
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *fancyhandler) appendLevel(buf *buffer, level slog.Level) {
	switch {
	case level < slog.LevelInfo:
		buf.WriteString("DBG")
		appendLevelDelta(buf, level-slog.LevelDebug)
	case level < slog.LevelWarn:
		buf.WriteStringIf(!h.noColor, ansiBrightGreen)
		buf.WriteString("INF")
		appendLevelDelta(buf, level-slog.LevelInfo)
		buf.WriteStringIf(!h.noColor, ansiReset)
	case level < slog.LevelError:
		buf.WriteStringIf(!h.noColor, ansiBrightYellow)
		buf.WriteString("WRN")
		appendLevelDelta(buf, level-slog.LevelWarn)
		buf.WriteStringIf(!h.noColor, ansiReset)
	default:
		buf.WriteStringIf(!h.noColor, ansiBrightRed)
		buf.WriteString("ERR")
		appendLevelDelta(buf, level-slog.LevelError)
		buf.WriteStringIf(!h.noColor, ansiReset)
	}
}

func appendLevelDelta(buf *buffer, delta slog.Level) {
	if delta == 0 {
		return
	} else if delta > 0 {
		buf.WriteByte('+')
	}
	*buf = strconv.AppendInt(*buf, int64(delta), 10)
}

func (h *fancyhandler) appendSource(buf *buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)

	buf.WriteStringIf(!h.noColor, ansiFaint)
	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	buf.WriteString(strconv.Itoa(src.Line))
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *fancyhandler) appendAttr(buf *buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	attr.Value = attr.Value.Resolve()
	if rep := h.replaceAttr; rep != nil && attr.Value.Kind() != slog.KindGroup {
		attr = rep(groups, attr)
		attr.Value = attr.Value.Resolve()
	}

	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
		return
	}

	switch attr.Key {
	case "err", "error.message":
		h.appendError(buf, attr.Key, attr.Value, groupsPrefix)
	case "status", "http.status":
		h.appendHttpStatus(buf, attr.Key, attr.Value, groupsPrefix)
	case "stacktrace", "error.stack":
		h.appendStackTrace(buf, attr.Key, attr.Value)
	default:
		h.appendKey(buf, attr.Key, groupsPrefix)
		h.appendValue(buf, attr.Value, true)
	}

	buf.WriteByte(' ')
}

func (h *fancyhandler) appendKey(buf *buffer, key, groups string) {
	buf.WriteStringIf(!h.noColor, ansiFaint)
	appendString(buf, groups+key, true)
	buf.WriteByte('=')
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *fancyhandler) appendValue(buf *buffer, v slog.Value, quote bool) {
	switch v.Kind() {
	case slog.KindString:
		appendString(buf, v.String(), quote)
	case slog.KindInt64:
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
	case slog.KindUint64:
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
	case slog.KindFloat64:
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'g', -1, 64)
	case slog.KindBool:
		*buf = strconv.AppendBool(*buf, v.Bool())
	case slog.KindDuration:
		appendString(buf, v.Duration().String(), quote)
	case slog.KindTime:
		appendString(buf, v.Time().String(), quote)
	case slog.KindAny:
		switch cv := v.Any().(type) {
		case slog.Level:
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendString(buf, string(data), quote)
		case *slog.Source:
			h.appendSource(buf, cv)
		default:
			appendString(buf, fmt.Sprintf("%+v", v.Any()), quote)
		}
	}
}

func (h *fancyhandler) appendError(buf *buffer, key string, val slog.Value, groupsPrefix string) {
	buf.WriteStringIf(!h.noColor, ansiBrightRedFaint)
	appendString(buf, groupsPrefix+key, true)
	buf.WriteByte('=')
	buf.WriteStringIf(!h.noColor, ansiResetFaint)
	appendString(buf, val.String(), true)
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *fancyhandler) appendStackTrace(buf *buffer, key string, val slog.Value) {
	buf.WriteString("\n" + key + "\n")
	buf.WriteStringIf(!h.noColor, ansiFaint)
	appendString(buf, val.String()+"\n", false)
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func (h *fancyhandler) appendHttpStatus(buf *buffer, key string, val slog.Value, groupsPrefix string) {
	code := val.Int64()
	color := ansiFaint

	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		color = ansiBrightGreenFaint
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		color = ansiBrightYellowFaint
	case code >= http.StatusInternalServerError:
		color = ansiBrightRedFaint
	}

	buf.WriteStringIf(!h.noColor, color)
	appendString(buf, groupsPrefix+key, true)
	buf.WriteByte('=')
	buf.WriteStringIf(!h.noColor, ansiResetFaint)
	*buf = strconv.AppendInt(*buf, code, 10)
	buf.WriteStringIf(!h.noColor, ansiReset)
}

func appendString(buf *buffer, s string, quote bool) {
	if quote && needsQuoting(s) {
		*buf = strconv.AppendQuote(*buf, s)
	} else {
		buf.WriteString(s)
	}
}

func needsQuoting(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, r := range s {
		if unicode.IsSpace(r) || r == '"' || r == '=' || !unicode.IsPrint(r) {
			return true
		}
	}
	return false
}
