package frontend

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aexvir/skladka/internal/auth"
	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/frontend/layouts"
	"github.com/aexvir/skladka/internal/frontend/views"
	"github.com/aexvir/skladka/internal/logging"
	"github.com/aexvir/skladka/internal/paste"
)

//go:embed static/*
var static embed.FS

// DashboardRouter returns a chi.Router that handles all frontend routes.
// It sets up routes for static assets and implements the main application
// endpoints.
//
// The router uses the provided Storage implementation for paste operations
// and automatically handles template rendering and static asset serving.
func DashboardRouter(ctx context.Context, pastesvc *paste.Service, authsvc *auth.Service) chi.Router {
	router := chi.NewRouter()

	router.Use(authsvc.UserSessionMiddleware())

	staticsrv := http.FileServerFS(static)
	etag := fmt.Sprintf(`"%s"`, config.BuildRevision)
	router.Get("/static/*",
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-None-Match") == etag {
					w.WriteHeader(http.StatusNotModified)
					return
				}

				w.Header().Set("ETag", etag)
				w.Header().Set("Cache-Control", "public, max-age=31536000")

				staticsrv.ServeHTTP(w, r)
			},
		),
	)

	router.Get("/",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "rendering creation page")

				return layouts.Base(
					auth.UserFromContext(r.Context()),
					views.Creation("Skládka"),
				).Render(r.Context(), w)
			},
		),
	)

	router.Post("/",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.dashboard", "creating paste")

				if err := r.ParseForm(); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "error parsing form", err)
				}

				var password *string
				if value := r.FormValue("password"); value != "" {
					password = &value
				}

				var tags []string
				if value := r.FormValue("tags"); value != "" {
					tags = strings.Split(value, ",")
				}

				paste, err := pastesvc.CreatePaste(r.Context(),
					auth.UserFromContext(r.Context()),
					r.FormValue("title"),
					r.FormValue("content"),
					r.FormValue("syntax"),
					r.FormValue("unlisted") != "on",
					tags,
					password,
					r.FormValue("expiration"),
				)
				if err != nil {
					return errors.NewHTTPError(http.StatusInternalServerError, "error creating paste", err)
				}

				http.Redirect(w, r, fmt.Sprintf("/p/%s", paste.Reference), http.StatusSeeOther)
				return nil
			},
		),
	)

	router.Get("/archive",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				logger := logging.FromContext(r.Context())
				logger.Info("frontend.archive", "rendering archive page")

				user := auth.UserFromContext(r.Context())
				username := r.URL.Query().Get("username")

				var (
					pastes []paste.Paste
					err    error
				)

				if username != "" {
					pastes, err = pastesvc.ListUserPastes(r.Context(), user, username)
				} else {
					pastes, err = pastesvc.ListPastes(r.Context(), user)
				}

				if err != nil {
					logger.Error(err, "frontend.archive", "error listing pastes")
					return errors.AsHTTPError(err)
				}

				return layouts.Base(
					user, views.Archive("Recent Pastes", pastes),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/p/{ref}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				user := auth.UserFromContext(r.Context())

				paste, err := pastesvc.GetPaste(r.Context(), user, ref)
				if err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error fetching paste %s", ref),
						err,
					)
				}

				if paste.Password != nil {
					return layouts.Base(
						user, views.PasswordPrompt(ref),
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
					user, views.Document(user, paste),
				).Render(r.Context(), w)
			},
		),
	)

	router.Post("/p/{ref}/unlock",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				password := r.FormValue("password")
				user := auth.UserFromContext(r.Context())

				paste, err := pastesvc.GetPasteWithPassword(r.Context(), user, ref, password)
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
					user, views.Document(user, *paste),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/r/{ref}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				password := r.Header.Get("x-skd-password")
				user := auth.UserFromContext(r.Context())

				paste, err := pastesvc.GetPaste(r.Context(), user, ref)
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
							user, views.RawPasswordPrompt(ref),
						).Render(r.Context(), w)
					}

					paste, err := pastesvc.GetPasteWithPassword(r.Context(), user, ref, password)
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

	router.Delete(
		"/p/{ref}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.dashboard", "deleting paste",
						"ref", ref,
					)

				ctx := r.Context()
				if err := pastesvc.DeletePaste(ctx, auth.UserFromContext(ctx), ref); err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error deleting paste %s", ref),
						err,
					)
				}

				http.Redirect(w, r, "/", http.StatusSeeOther)
				return nil
			},
		),
	)

	router.Get("/u/profile",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				user := auth.UserFromContext(r.Context())

				if user == nil {
					http.Redirect(w, r, fmt.Sprintf("/u/login"), http.StatusSeeOther)
					return nil
				}

				return layouts.Base(
					user, views.Profile(*user, user),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/u/{username}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				user := auth.UserFromContext(r.Context())

				profile, err := authsvc.GetUser(r.Context(), chi.URLParam(r, "username"))
				if err != nil {
					return errors.NewHTTPError(
						http.StatusUnprocessableEntity,
						fmt.Sprintf("error getting user %s", chi.URLParam(r, "username")),
						err,
					)
				}

				return layouts.Base(
					user, views.Profile(profile, user),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/u/login",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				if user := auth.UserFromContext(r.Context()); user != nil {
					http.Redirect(w, r, fmt.Sprintf("/u/profile"), http.StatusSeeOther)
					return nil
				}

				return layouts.Base(
					nil, views.Login(),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/u/logout",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				cookie, err := r.Cookie("session")
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "missing session cookie", err)
				}

				session, err := authsvc.GetSession(r.Context(), cookie.Value)
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "session not found", err)
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    session.Token,
					Path:     "/",
					Expires:  time.Now().Add(-1 * time.Hour),
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				http.Redirect(w, r, fmt.Sprintf("/u/login"), http.StatusSeeOther)
				return nil
			},
		),
	)

	router.Post("/u/login/exists",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				var req struct {
					Username string `json:"username"`
				}

				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "invalid request body", err)
				}
				defer r.Body.Close()

				_, err := authsvc.GetUser(r.Context(), req.Username)
				if err != nil {
					return errors.NewHTTPError(http.StatusNotFound, "user not found", nil)
				}

				w.WriteHeader(http.StatusOK)
				return nil
			},
		),
	)

	router.Post("/u/login/start",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				var req struct {
					Username string `json:"username"`
				}

				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "invalid request body", err)
				}
				defer r.Body.Close()

				options, session, err := authsvc.BeginLogin(r.Context(), req.Username)
				if err != nil {
					return err
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    session.Token,
					Path:     "/",
					Expires:  session.ExpiresAt,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				w.Header().Set("Content-Type", "application/json")
				return json.NewEncoder(w).Encode(options)
			},
		),
	)

	router.Post("/u/login/finish",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				cookie, err := r.Cookie("session")
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "missing session cookie", err)
				}

				session, err := authsvc.FinishLogin(ctx, cookie.Value, r)
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "error during login", err)
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    session.Token,
					Path:     "/",
					Expires:  session.ExpiresAt,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				w.WriteHeader(http.StatusOK)
				return nil
			},
		),
	)

	router.Post("/u/register/start",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				var req struct {
					Username string `json:"username"`
				}

				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "invalid request body", err)
				}
				defer r.Body.Close()

				options, session, err := authsvc.BeginRegister(r.Context(), req.Username)
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "error during registration", err)
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    session.Token,
					Path:     "/",
					Expires:  session.ExpiresAt,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				w.Header().Set("Content-Type", "application/json")
				return json.NewEncoder(w).Encode(options)
			},
		),
	)

	router.Post("/u/register/finish",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				cookie, err := r.Cookie("session")
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "missing session cookie", err)
				}

				session, err := authsvc.FinishRegister(r.Context(), cookie.Value, r)
				if err != nil {
					return errors.NewHTTPError(http.StatusBadRequest, "error during registration", err)
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "session",
					Value:    session.Token,
					Path:     "/",
					Expires:  session.ExpiresAt,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
				})

				w.WriteHeader(http.StatusOK)
				return nil
			},
		),
	)

	return router
}
