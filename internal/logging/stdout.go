package logging

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/log"
	sdk "go.opentelemetry.io/otel/sdk/log"
)

// stdOutProcessor implements sdk.Processor to write logs to stdout using fancy formatting
type stdOutProcessor struct {
	handler slog.Handler
}

// NewStdOutProcessor creates a new processor that writes logs to stdout using fancy formatting
func NewStdOutProcessor(handler slog.Handler) sdk.Processor {
	return &stdOutProcessor{
		handler: handler,
	}
}

func (p *stdOutProcessor) OnEmit(ctx context.Context, record *sdk.Record) error {
	slogrec := slog.Record{
		Time:    record.Timestamp(),
		Message: record.Body().AsString(),
	}

	record.WalkAttributes(
		func(attr log.KeyValue) bool {
			switch attr.Value.Kind() {
			case log.KindBool:
				slogrec.AddAttrs(slog.Bool(attr.Key, attr.Value.AsBool()))
			case log.KindFloat64:
				slogrec.AddAttrs(slog.Float64(attr.Key, attr.Value.AsFloat64()))
			case log.KindInt64:
				slogrec.AddAttrs(slog.Int64(attr.Key, attr.Value.AsInt64()))
			case log.KindString:
				slogrec.AddAttrs(slog.String(attr.Key, attr.Value.AsString()))
			default:
				slogrec.AddAttrs(slog.Any(attr.Key, attr.Value))
			}
			return true
		},
	)

	return p.handler.Handle(ctx, slogrec)
}

func (p *stdOutProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *stdOutProcessor) ForceFlush(ctx context.Context) error {
	return nil
}
