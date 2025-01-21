package frontend

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aexvir/skladka/internal/attributes"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/frontend/layouts"
	"github.com/aexvir/skladka/internal/frontend/views"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/metrics"
	"github.com/aexvir/skladka/internal/paste"
)

// Storage defines the interface for paste storage operations required by the frontend.
// This interface allows the frontend to be decoupled from the actual storage implementation,
// making it easier to test and maintain.
type Storage interface {
	// GetPaste retrieves a paste by its reference.
	GetPaste(context.Context, string) (paste.Paste, error)

	// GetPasteWithPassword retrieves a paste by its reference.
	GetPasteWithPassword(context.Context, string, string) (*paste.Paste, error)

	// CreatePaste stores a new paste and returns its reference.
	CreatePaste(context.Context, paste.Paste) (string, error)

	// ListPastes returns all public pastes.
	ListPastes(context.Context) ([]paste.Paste, error)
}

//go:embed static/*
var static embed.FS

// DashboardRouter returns a chi.Router that handles all frontend routes.
// It sets up routes for static assets and implements the main application
// endpoints.
//
// The router uses the provided Storage implementation for paste operations
// and automatically handles template rendering and static asset serving.
func DashboardRouter(ctx context.Context, storage Storage) chi.Router {
	router := chi.NewRouter()

	met := new(Metrics)
	if err := metrics.FromContext(ctx).Register(met); err != nil {
		panic(errors.Wrap(err, "error initializing frontend metrics"))
	}

	staticsrv := http.FileServerFS(static)
	router.Get(
		"/static/*",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
				// w.Header().Set("Expires", time.Now().Add(time.Hour*24*365).UTC().Format(http.TimeFormat))
				// w.Header().Set("Pragma", "public")

				staticsrv.ServeHTTP(w, r)
			},
		),
	)

	router.Get(
		"/",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "rendering creation page")

				return layouts.Base(
					views.Creation("Skládka"),
				).Render(r.Context(), w)
			},
		),
	)

	router.Post(
		"/",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "creating paste")

				if err := r.ParseForm(); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "error parsing form", err)
				}

				// Create paste object
				p := paste.Paste{
					Title:   r.FormValue("title"),
					Content: r.FormValue("content"),
					Syntax:  r.FormValue("syntax"),
					Public:  r.FormValue("unlisted") != "on",
				}

				if tags := r.FormValue("tags"); tags != "" {
					p.Tags = strings.Split(tags, ",")
				}

				if password := r.FormValue("password"); password != "" {
					p.Password = &password
				}

				if expiration := r.FormValue("expiration"); expiration != "" {
					delta, err := time.ParseDuration(expiration)
					if err != nil {
						return errors.NewHTTPError(http.StatusBadRequest, "invalid expiration value", err)
					}
					deadline := time.Now().Add(delta)
					p.Expiration = &deadline
				}

				if err := p.Validate(); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "invalid paste", err)
				}

				// Save to storage
				ref, err := storage.CreatePaste(r.Context(), p)
				if err != nil {
					return errors.NewHTTPError(http.StatusInternalServerError, "error creating paste", err)
				}

				met.PasteCreations.Add(r.Context(), 1, attributes.Status("ok"))
				met.PasteSize.Record(r.Context(), int(p.SizeBytes()))

				// Redirect to the paste view
				http.Redirect(w, r, fmt.Sprintf("/%s", ref), http.StatusSeeOther)
				return nil
			},
		),
	)

	router.Get(
		"/archive",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.archive", "rendering archive page")

				pastes, err := storage.ListPastes(r.Context())
				if err != nil {
					logger.Error(err, "frontend.archive", "error listing pastes")
					return errors.AsHTTPError(err)
				}

				met.PasteRetrievals.Add(r.Context(), len(pastes), attributes.Status(attributes.ValueStatusOk))

				return layouts.Base(
					views.Archive("Recent Pastes", pastes),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get(
		"/{ref}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")

				paste, err := storage.GetPaste(r.Context(), ref)
				if err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error fetching paste %s", ref),
						err,
					)
				}

				if paste.Password != nil {
					return layouts.Base(
						views.PasswordPrompt(ref),
					).Render(r.Context(), w)
				}

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.dashboard", "rendering document page",
						"ref", ref,
						"title", paste.Title,
						"syntax", paste.Syntax,
						"tags", paste.Tags,
					)

				return layouts.Base(
					views.Document(paste),
				).Render(r.Context(), w)
			},
		),
	)

	router.Post("/{ref}/unlock",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				password := r.FormValue("password")

				paste, err := storage.GetPasteWithPassword(r.Context(), ref, password)
				if err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error fetching paste %s", ref),
						err,
					)
				}

				if paste == nil {
					return errors.NewHTTPError(http.StatusForbidden, "invalid password", err)
				}

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.dashboard", "rendering document page",
						"ref", ref,
						"title", paste.Title,
						"syntax", paste.Syntax,
						"tags", paste.Tags,
					)

				return layouts.Base(
					views.Document(*paste),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get(
		"/{ref}/raw",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				password := r.Header.Get("x-skd-password")

				paste, err := storage.GetPaste(r.Context(), ref)
				if err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error fetching paste %s", ref),
						err,
					)
				}

				if paste.Password != nil {
					if password == "" {
						return layouts.Base(
							views.RawPasswordPrompt(ref),
						).Render(r.Context(), w)
					}

					paste, err := storage.GetPasteWithPassword(r.Context(), ref, password)
					if err != nil {
						return errors.NewHTTPError(
							http.StatusUnprocessableEntity,
							fmt.Sprintf("error fetching paste %s", ref),
							err,
						)
					}

					if paste == nil {
						return errors.NewHTTPError(http.StatusForbidden, "invalid password", err)
					}
				}

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.dashboard", "rendering raw document page",
						"ref", ref,
						"title", paste.Title,
						"syntax", paste.Syntax,
						"tags", paste.Tags,
					)

				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, err = w.Write([]byte(paste.Content))
				return err
			},
		),
	)

	return router
}
