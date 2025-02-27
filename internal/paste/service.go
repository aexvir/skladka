package paste

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aexvir/skladka/internal/attributes"
	"github.com/aexvir/skladka/internal/auth"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/tracing"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	store         Storage
	met           *Metrics
	encryptionkey string
}

type Storage interface {
	// GetPaste retrieves a paste by its reference.
	GetPaste(context.Context, string) (Paste, error)

	// GetPasteWithPassword retrieves a paste by its reference.
	GetPasteWithPassword(context.Context, string, string) (*Paste, error)

	// DeletePaste deletes a paste by its reference.
	DeletePaste(context.Context, string) error

	// CreatePaste stores a new paste and returns its reference.
	CreatePaste(context.Context, Paste) (string, error)

	// ListPastes returns all public pastes.
	ListPastes(context.Context) ([]Paste, error)

	// ListUserPastes returns all pastes created by a specific user
	ListUserPastes(context.Context, string) ([]Paste, error)
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

func (svc *Service) GenerateSignedSecret(paste Paste, expiration time.Duration) (string, int64) {
	deadline := time.Now().Add(expiration).Unix()

	h := hmac.New(sha512.New, []byte(svc.encryptionkey))
	h.Write(
		fmt.Appendf(nil,
			"ref=%s;created=%d;deadline=%d",
			paste.Reference, paste.Creation.Unix(), deadline,
		),
	)

	return hex.EncodeToString(h.Sum(nil)), deadline
}

func (svc *Service) VerifySignature(paste Paste, signature string, deadline int64) bool {
	if time.Now().Unix() > int64(deadline) {
		return false
	}

	h := hmac.New(sha512.New, []byte(svc.encryptionkey))
	h.Write(
		fmt.Appendf(nil,
			"ref=%s;created=%d;deadline=%d",
			paste.Reference, paste.Creation.Unix(), deadline,
		),
	)

	expected := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
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
	mimetype *string,
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
		Mimetype: mimetype,
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

// DeletePaste deletes a paste by its reference and owner username.
func (svc *Service) DeletePaste(ctx context.Context, user *auth.User, ref string) (err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.DeletePaste")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteDeletions.Add(ctx, 1, attributes.Status(status))
	}()

	paste, err := svc.store.GetPaste(ctx, ref)
	if err != nil {
		return errors.Wrap(err, "error retrieving paste")
	}

	if paste.Owner == nil {
		// will be deletable by admins later on, for now, reject
		return errors.New("anonymous pastes cannot be deleted")
	}

	if user == nil || (*paste.Owner != user.Username) {
		return errors.New("unauthorized")
	}

	return svc.store.DeletePaste(ctx, ref)
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

// ListUserPastes retrieves all pastes for a specific user from the storage.
// The function returns a slice of Paste objects and any error that occurred during the operation.
func (svc *Service) ListUserPastes(ctx context.Context, user *auth.User, username string) (pastes []Paste, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "paste.Service.ListUserPastes")
	defer func() {
		finish(&err)
		status := attributes.ValueStatusOk
		if err != nil {
			status = attributes.ValueStatusError
		}
		svc.met.PasteRetrievals.Add(ctx, len(pastes), attributes.Status(status))
	}()

	pastes, err = svc.store.ListUserPastes(ctx, username)
	if err != nil {
		return nil, errors.Wrap(err, "error retrieving user pastes")
	}

	// the req is authenticated and the user is asking for their own pastes
	// no filtering needed
	if user != nil && username == user.Username {
		return pastes, nil
	}

	// otherwise only public pastes are visible to other users
	filtered := make([]Paste, 0, len(pastes))
	for _, paste := range pastes {
		if paste.Public {
			filtered = append(filtered, paste)
		}
	}

	return filtered, nil
}
