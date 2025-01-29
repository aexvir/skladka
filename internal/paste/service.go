package paste

import (
	"context"
	"time"

	"github.com/aexvir/skladka/internal/attributes"
	"github.com/aexvir/skladka/internal/auth"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/tracing"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	store Storage
	met   *Metrics
}

type Storage interface {
	// GetPaste retrieves a paste by its reference.
	GetPaste(context.Context, string) (Paste, error)

	// GetPasteWithPassword retrieves a paste by its reference.
	GetPasteWithPassword(context.Context, string, string) (*Paste, error)

	// CreatePaste stores a new paste and returns its reference.
	CreatePaste(context.Context, Paste) (string, error)

	// ListPastes returns all public pastes.
	ListPastes(context.Context) ([]Paste, error)
}

// NewService creates a new paste service with the provided storage.
func NewService(ctx context.Context, store Storage) (*Service, error) {
	met := new(Metrics)
	if err := metrics.FromContext(ctx).Register(met); err != nil {
		return nil, errors.Wrap(err, "error initializing paste metrics")
	}

	return &Service{
		store: store,
		met:   met,
	}, nil
}

// CreatePaste creates a new paste with the given parameters.
// Returns a [Paste] instance with its reference populated.
func (svc *Service) CreatePaste(
	ctx context.Context,
	user *auth.User,
	title, content, syntax string,
	public bool,
	tags []string,
	password *string,
	expiration string,
) (paste Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.CreatePaste")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteCreations.Add(ctx, 1, attributes.Status(status))
		svc.met.PasteSize.Record(ctx, int(paste.SizeBytes()))
	}()

	paste = Paste{
		Title:    title,
		Content:  content,
		Syntax:   syntax,
		Public:   public,
		Tags:     tags,
		Password: password,
	}

	if user != nil {
		paste.Owner = &user.Username
	}

	if expiration != "no expiration" {
		delta, err := time.ParseDuration(expiration)
		if err != nil {
			return paste, errors.Wrap(err, "invalid expiration value")
		}
		deadline := time.Now().Add(delta)
		paste.Expiration = &deadline
	}

	if err := paste.Validate(); err != nil {
		return paste, errors.Wrap(err, "invalid paste")
	}

	paste.Reference, err = svc.store.CreatePaste(ctx, paste)
	return paste, errors.Wrap(err, "error storing paste")
}

// GetPaste retrieves a paste by its reference.
func (svc *Service) GetPaste(ctx context.Context, _ *auth.User, ref string) (paste Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.GetPaste")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteRetrievals.Add(ctx, 1, attributes.Status(status))
	}()

	return svc.store.GetPaste(ctx, ref)
}

// GetPasteWithPassword retrieves a password-protected paste by its reference.
func (svc *Service) GetPasteWithPassword(ctx context.Context, _ *auth.User, ref, password string) (paste *Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.GetPasteWithPassword")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteRetrievals.Add(ctx, 1, attributes.Status(status))
	}()

	return svc.store.GetPasteWithPassword(ctx, ref, password)
}

// ListPastes retrieves all public pastes from the storage.
// The function returns a slice of Paste objects and any error that occurred during the operation.
func (svc *Service) ListPastes(ctx context.Context, _ *auth.User) (pastes []Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.ListPastes")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteRetrievals.Add(ctx, len(pastes), attributes.Status(status))
	}()

	return svc.store.ListPastes(ctx)
}
