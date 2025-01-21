package attributes

import (
	"go.opentelemetry.io/otel/attribute"
)

const (
	KeyStatus = "status"

	ValueStatusOk    = "ok"
	ValueStatusError = "error"
)

func Status(value string) attribute.KeyValue {
	return String(KeyStatus, value)
}
