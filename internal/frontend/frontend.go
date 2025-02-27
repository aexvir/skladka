package frontend

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

				// Parse multipart form with 50MB max memory
				if err := r.ParseMultipartForm(50 << 20); err != nil {
					// If not a multipart form, try to parse regular form
					if err := r.ParseForm(); err != nil {
						return errors.NewHTTPError(http.StatusBadRequest, "error parsing form", err)
					}
				}

				var password *string
				if value := r.FormValue("password"); value != "" {
					password = &value
				}

				var tags []string
				if value := r.FormValue("tags"); value != "" {
					tags = strings.Split(value, ",")
				}

				content := r.FormValue("content")
				syntax := r.FormValue("syntax")

				var mimetype *string
				if mime := r.FormValue("mimetype"); mime != "" {
					mimetype = &mime

					if r.Form.Has("file") {
						file, _, err := r.FormFile("file")
						if err == nil {
							defer file.Close()

							rawdata, err := io.ReadAll(file)
							if err != nil {
								return errors.NewHTTPError(http.StatusInternalServerError, "error reading uploaded file", err)
							}

							content = string(rawdata)
						}
					}
				}

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.create", "creating paste",
						"title", r.FormValue("title"),
						"syntax", syntax,
						"hasContent", content != "",
						"contentLength", len(content),
						"mimetype", mimetype,
					)

				paste, err := pastesvc.CreatePaste(r.Context(),
					auth.UserFromContext(r.Context()),
					r.FormValue("title"),
					content,
					syntax,
					r.FormValue("unlisted") != "on",
					tags,
					password,
					r.FormValue("expiration"),
					mimetype,
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

				signature, deadline := pastesvc.GenerateSignedSecret(paste, 10*time.Second)
				return layouts.Base(
					user, views.Document(user, paste, signature, deadline),
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

				signature, deadline := pastesvc.GenerateSignedSecret(*paste, 10*time.Second)
				return layouts.Base(
					user, views.Document(user, *paste, signature, deadline),
				).Render(r.Context(), w)
			},
		),
	)

	router.Get("/r/{ref}",
		errors.WithErrorHandler(
			func(w http.ResponseWriter, r *http.Request) error {
				ref := chi.URLParam(r, "ref")
				signature := r.URL.Query().Get("signature")
				deadline := r.URL.Query().Get("deadline")
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
					switch {
					case password != "": // password provided, attempt to unlock with it
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

					case signature != "": // signed url, verify validity
						deadlinesec, err := strconv.ParseInt(deadline, 10, 64)
						if err != nil || !pastesvc.VerifySignature(paste, signature, deadlinesec) {
							http.Redirect(w, r, fmt.Sprintf("/r/%s", paste.Reference), http.StatusSeeOther)
							return nil
						}

					default: // missing any kind of secret, prompt for password
						return layouts.Base(
							user, views.RawPasswordPrompt(ref),
						).Render(r.Context(), w)
					}
				}

				logging.
					FromContext(r.Context()).
					Info(
						"frontend.dashboard", "rendering raw document",
						"ref", ref,
						"title", paste.Title,
						"syntax", paste.Syntax,
						"tags", paste.Tags,
						"mimetype", paste.Mimetype,
						"size", paste.SizeBytes,
					)

				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				if paste.Mimetype != nil {
					w.Header().Set("Content-Type", *paste.Mimetype)
				}
				// w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, paste.FileName()))

				content := []byte(paste.Content)

				if paste.IsBase64Encoded() {
					signature, deadline := pastesvc.GenerateSignedSecret(paste, 5*time.Minute)
					w.Header().Set("x-skd-signature", signature)
					w.Header().Set("x-skd-deadline", strconv.FormatInt(deadline, 10))

					decoded, err := base64.StdEncoding.DecodeString(paste.Content)
					if err != nil {
						return errors.NewHTTPError(
							http.StatusUnprocessableEntity,
							fmt.Sprintf("error base64 decoding paste %s", ref),
							err,
						)
					}
					content = decoded
				}

				_, err = w.Write(content)
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
