package auth

import (
	"context"
	"net/http"
)

const ctxKeyUser = "user"

// UserSessionMiddleware populates the context with a reference
// to the user that's currently performing the requests.
// This is only done if:
//   - there's a session cookie
//   - the session exists in the database
//   - the session is not expired
//   - the user with that username exists
//
// It doesn't prevent in any way the request from going through.
// In order to retrieve the populated user from the context, use [UserFromContext].
func (svc *Service) UserSessionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {

				// important!
				//
				token, err := r.Cookie("session")
				if err == nil { // if the cookie is set
					session, err := svc.store.GetSessionByToken(r.Context(), token.Value)
					if err == nil { // and the session can be retrieved
						if !session.IsExpired() { // and is not expired
							if !session.Throwaway { // and is not a throwaway session created during webauthn handshake
								user, err := svc.store.GetUserByUsername(r.Context(), session.Username)
								if err == nil {
									r = r.WithContext(
										context.WithValue(r.Context(), ctxKeyUser, &user),
									)
								}
							}
						}
					}
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}

// UserFromContext returns the user reference from the context, if it exists.
func UserFromContext(ctx context.Context) *User {
	val := ctx.Value(ctxKeyUser)
	if user, ok := val.(*User); ok {
		return user
	}
	return nil
}
