package frontend

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/aexvir/skladka/internal/frontend/layouts"
	"github.com/aexvir/skladka/internal/frontend/views"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/paste"
)

// Storage defines the interface for paste storage operations required by the frontend.
// This interface allows the frontend to be decoupled from the actual storage implementation,
// making it easier to test and maintain.
type Storage interface {
	// GetPaste retrieves a paste by its reference.
	GetPaste(context.Context, string) (paste.Paste, error)

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
func DashboardRouter(storage Storage) chi.Router {
	router := chi.NewRouter()

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
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "rendering creation page")

				layouts.Base(
					views.Creation("Skládka"),
				).Render(r.Context(), w)

				return
			},
		),
	)

	router.Post(
		"/",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "creating paste")

				if err := r.ParseForm(); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("error parsing form: %v", err)))
					return
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

				if err := p.Validate(); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(fmt.Sprintf("error creating paste: %v", err)))
					return
				}

				// Save to storage
				ref, err := storage.CreatePaste(r.Context(), p)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(fmt.Sprintf("error creating paste: %v", err)))
					return
				}

				// Redirect to the paste view
				http.Redirect(w, r, fmt.Sprintf("/%s", ref), http.StatusSeeOther)
				return
			},
		),
	)

	router.Get(
		"/archive",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.archive", "rendering archive page")

				pastes, err := storage.ListPastes(r.Context())
				if err != nil {
					logger.Error(err, "frontend.archive", "error listing pastes")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				layouts.Base(
					views.Archive("Recent Pastes", pastes),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get(
		"/{ref}",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ref := chi.URLParam(r, "ref")

				paste, err := storage.GetPaste(r.Context(), ref)
				if err != nil {
					w.WriteHeader(422)
					w.Write([]byte(fmt.Sprintf("error fetching paste %s: %v", ref, err)))
					return
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

				layouts.Base(
					views.Document(paste),
				).Render(r.Context(), w)
				return
			},
		),
	)

	return router
}
