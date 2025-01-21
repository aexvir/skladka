package frontend

import (
	"github.com/aexvir/skladka/internal/metrics"
)

type Metrics struct {
	// PasteCreations counts the number of pastes created
	PasteCreations metrics.IntCounter `metric:"paste.creations,number of pastes created"`

	// PasteSize tracks the size of created pastes in bytes
	PasteSize metrics.IntHistogram `metric:"paste.size,paste content size,bytes"`

	// PasteRetrievals counts the number of paste retrievals
	PasteRetrievals metrics.IntCounter `metric:"paste.retrievals,number of pastes retrieved"`
}
