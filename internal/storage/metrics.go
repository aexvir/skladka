package storage

import (
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the metrics for the storage package.
type Metrics struct {
	// PasteCreated counts the number of pastes created
	PasteCreated metric.Int64Counter `metric:"storage_paste_created_total,Number of pastes created"`

	// PasteSize tracks the size of created pastes in bytes
	PasteSize metric.Int64Histogram `metric:"storage_paste_size_bytes,Size of pastes in bytes"`

	// PasteRetrieved counts the number of paste retrievals
	PasteRetrieved metric.Int64Counter `metric:"storage_paste_retrieved_total,Number of pastes retrieved"`

	// PasteNotFound counts the number of paste retrieval attempts that resulted in not found
	PasteNotFound metric.Int64Counter `metric:"storage_paste_not_found_total,Number of paste retrieval attempts that resulted in not found"`

	// PasteErrors counts the number of errors encountered during paste operations
	PasteErrors metric.Int64Counter `metric:"storage_paste_errors_total,Number of errors encountered during paste operations"`
}
