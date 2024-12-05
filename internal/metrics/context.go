package metrics

import (
	"context"
)

const ctxKeyMeter = "meter"

// NewContext returns a new context.Context that carries the provided meter instance.
// This context should be used as the parent context for all operations that need
// to record metrics. The meter can be retrieved using FromContext.
func NewContext(parent context.Context, metrics *Meter) context.Context {
	return context.WithValue(parent, ctxKeyMeter, metrics)
}

// FromContext returns the Meter stored in ctx if it exists, or a no-op metrics instance.
// The no-op instance will silently discard all metrics but still allow metric registration
// to work without any conditional logic in the code.
//
// This function should be used to obtain the meter instance in any function that needs
// to record metrics. It will never return nil, making it safe to use without checks.
func FromContext(ctx context.Context) *Meter {
	meter, ok := ctx.Value(ctxKeyMeter).(*Meter)

	if meter == nil || !ok {
		return NewNoopMeter()
	}

	return meter
}
