package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/aexvir/skladka/internal/config"
	"github.com/aexvir/skladka/internal/errors"
	"github.com/aexvir/skladka/internal/tracing"
)

type Service struct {
	store    Storage
	webauthn *webauthn.WebAuthn
}

type Storage interface {
	CreateUser(context.Context, User) error
	UpdateUserCredentials(context.Context, User) error
	GetSessionByToken(context.Context, string) (Session, error)
	CreateSession(context.Context, Session) error
	GetUserByUsername(context.Context, string) (User, error)
}

// NewService creates a new authentication service with the provided storage and WebAuthn configuration.
// It initializes the WebAuthn relying party with the specified server ID, display name and origins.
// Returns the configured service instance and any error encountered during initialization.
func NewService(ctx context.Context, store Storage, cfg config.WebAuthn) (*Service, error) {
	w, err := webauthn.New(
		&webauthn.Config{
			RPID:          cfg.RelyingPartyServerID,
			RPDisplayName: cfg.RelyingPartyDisplayName,
			RPOrigins:     []string{cfg.RelyingPartyServerOrigin},
		},
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create webauthn service")
	}

	return &Service{
		store:    store,
		webauthn: w,
	}, nil
}

// GetUser retrieves a user from storage by their username. It returns the User object
// and any error encountered during the retrieval.
func (svc *Service) GetUser(ctx context.Context, username string) (user User, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.GetUser")
	defer finish(&err)

	return svc.store.GetUserByUsername(ctx, username)
}

// GetUser retrieves a session from storage by its token. It returns the Session object
// and any error encountered during the retrieval.
func (svc *Service) GetSession(ctx context.Context, token string) (session Session, err error) {
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.GetSession")
	defer finish(&err)

	return svc.store.GetSessionByToken(ctx, token)
}

// BeginRegister initiates the WebAuthn registration process for a new user.
// It checks if the username is available, creates a new WebAuthn credential creation challenge,
// and stores the registration session data.
//
// Returns:
//   - *protocol.CredentialCreation: WebAuthn credential creation options
//   - Session: The registration session containing challenge data
//   - error: Any error encountered during the process
//
// The method will return an error if:
//   - The username is already taken
//   - WebAuthn registration initialization fails
//   - Session creation fails
func (svc *Service) BeginRegister(ctx context.Context, username string) (*protocol.CredentialCreation, Session, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.BeginRegister")
	defer finish(&err)

	if _, err := svc.GetUser(ctx, username); err == nil {
		return nil, Session{}, errors.New("user already exists")
	}

	opts, data, err := svc.webauthn.BeginRegistration(
		&User{Username: username, UUID: uuid.NewString()},
		webauthn.WithAuthenticatorSelection(
			protocol.AuthenticatorSelection{
				RequireResidentKey: protocol.ResidentKeyRequired(),
				UserVerification:   protocol.VerificationRequired,
			},
		),
	)
	if err != nil {
		return nil, Session{}, errors.Wrap(err, "error starting webauthn registration process")
	}

	var sessdata bytes.Buffer
	if err := json.NewEncoder(&sessdata).Encode(data); err != nil {
		return nil, Session{}, errors.Wrap(err, "error encoding session data")
	}

	session := Session{
		Username:  username,
		Token:     uuid.NewString(),
		Data:      sessdata.Bytes(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := svc.store.CreateSession(ctx, session); err != nil {
		return nil, Session{}, errors.Wrap(err, "error storing temporary session")
	}

	return opts, session, nil
}

// FinishRegister completes the WebAuthn registration process for a new user.
// It validates the registration session, processes the WebAuthn credential response,
// and creates a new authenticated user session.
//
// Returns:
//   - *Session: The newly created authenticated session
//   - error: Any error encountered during registration
//
// The method will return an error if:
//   - The registration session is invalid or expired
//   - The WebAuthn credential validation fails
//   - User creation or session storage fails
func (svc *Service) FinishRegister(ctx context.Context, token string, response *http.Request) (*Session, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.FinishRegister")
	defer finish(&err)

	session, err := svc.GetSession(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, "session not found")
	}

	if session.IsExpired() {
		return nil, errors.New("session expired")
	}

	var data webauthn.SessionData
	if err := json.NewDecoder(bytes.NewReader(session.Data)).Decode(&data); err != nil {
		return nil, errors.Wrap(err, "error decoding session data")
	}

	user := User{
		UUID:     string(data.UserID),
		Username: session.Username,
	}

	credential, err := svc.webauthn.FinishRegistration(&user, data, response)
	if err != nil {
		return nil, errors.Wrap(err, "error finishing webauthn registration process")
	}

	user.Credentials = append(user.Credentials, *credential)

	authed := Session{
		Username:  user.Username,
		Token:     uuid.NewString(),
		Data:      []byte("{}"),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}

	if err := svc.store.CreateSession(ctx, authed); err != nil {
		return nil, errors.Wrap(err, "error storing auth session")
	}

	return &authed, svc.store.CreateUser(ctx, user)
}

// BeginLogin initiates the WebAuthn login process for an existing user.
// It retrieves the user's information, creates a WebAuthn assertion challenge,
// and stores the login session data.
//
// Returns:
//   - *protocol.CredentialAssertion: WebAuthn credential assertion options
//   - Session: The login session containing challenge data
//   - error: Any error encountered during the process
//
// The method will return an error if:
//   - The user does not exist
//   - WebAuthn login initialization fails
//   - Session data encoding fails
//   - Session creation fails
func (svc *Service) BeginLogin(ctx context.Context, username string) (*protocol.CredentialAssertion, Session, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.BeginLogin")
	defer finish(&err)

	user, err := svc.GetUser(ctx, username)
	if err != nil {
		return nil, Session{}, errors.Wrap(err, "user not found")
	}

	opts, data, err := svc.webauthn.BeginLogin(&user)
	if err != nil {
		return nil, Session{}, errors.Wrap(err, "error starting webauthn login process")
	}

	var sessdata bytes.Buffer
	if err := json.NewEncoder(&sessdata).Encode(data); err != nil {
		return nil, Session{}, errors.Wrap(err, "error encoding session data")
	}

	session := Session{
		Username:  user.Username,
		Token:     uuid.New().String(),
		Data:      sessdata.Bytes(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if err := svc.store.CreateSession(ctx, session); err != nil {
		return nil, Session{}, errors.Wrap(err, "error storing temporary session")
	}

	return opts, session, nil
}

// FinishLogin completes the WebAuthn login process for a user.
// It validates the login session, processes the WebAuthn credential response,
// and creates a new authenticated session.
//
// Returns:
//   - *Session: The newly created authenticated session
//   - error: Any error encountered during login
//
// The method will return an error if:
//   - The login session is invalid or expired
//   - The user cannot be found
//   - The WebAuthn credential validation fails
//   - User credential update fails
//   - Session creation fails
func (svc *Service) FinishLogin(ctx context.Context, token string, response *http.Request) (*Session, error) {
	var err error
	ctx, finish := tracing.FromContext(ctx, trace.SpanKindInternal, "auth.Service.FinishLogin")
	defer finish(&err)

	session, err := svc.GetSession(ctx, token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain session")
	}

	if session.IsExpired() {
		return nil, errors.New("session expired")
	}

	user, err := svc.store.GetUserByUsername(ctx, session.Username)
	if err != nil {
		return nil, errors.Wrap(err, "user not found")
	}

	var data webauthn.SessionData
	if err := json.NewDecoder(bytes.NewReader(session.Data)).Decode(&data); err != nil {
		return nil, errors.Wrap(err, "error decoding session data")
	}

	credential, err := svc.webauthn.FinishLogin(&user, data, response)
	if err != nil {
		return nil, errors.Wrap(err, "error finishing webauthn login process")
	}
	user.Credentials = append(user.Credentials, *credential)

	if err := svc.store.UpdateUserCredentials(ctx, user); err != nil {
		return nil, errors.Wrap(err, "error updating user credentials")
	}

	authed := Session{
		Username:  user.Username,
		Token:     uuid.NewString(),
		Data:      []byte("{}"),
		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
	}

	if err := svc.store.CreateSession(ctx, authed); err != nil {
		return nil, errors.Wrap(err, "error storing auth session")
	}

	return &authed, nil
}
