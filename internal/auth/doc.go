// Package auth provides WebAuthn-based authentication functionality for the application.
//
// The package implements user authentication using [WebAuthn] (Web Authentication),
// which allows for passwordless authentication using platform authenticators
// (like biometric sensors, security keys, etc).
//
// Key components:
//
//   - Service: The main service that handles WebAuthn operations and user management
//   - User: Implements the webauthn.User interface and represents an authenticated user
//   - Session: Represents a user's authentication session with expiration
//   - Storage: Interface defining the required persistence operations
//   - Middleware: HTTP middleware for user session management
//
// The authentication flow consists of two main operations:
//
//  1. Registration (BeginRegister/FinishRegister):
//     - Creates new users with WebAuthn credentials
//     - Requires resident keys and user verification
//     - Manages temporary sessions during the registration process
//
//  2. Login (BeginLogin/FinishLogin):
//     - Authenticates existing users using their WebAuthn credentials
//     - Creates and manages user sessions
//
// The package also provides a middleware that automatically populates
// the request context with the authenticated user information when
// a valid session is present.
//
// [WebAuthn]: https://webauthn.guide/
package auth
