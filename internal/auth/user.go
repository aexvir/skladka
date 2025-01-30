package auth

import (
	"github.com/go-webauthn/webauthn/webauthn"
)

// User represents a WebAuthn user.
// It implements the [webauthn.User] interface.
type User struct {
	UUID        string
	Username    string
	Credentials []webauthn.Credential
	Avatar      string
}

func (u *User) WebAuthnID() []byte {
	return []byte(u.UUID)
}

func (u *User) WebAuthnName() string {
	return u.Username
}

func (u *User) WebAuthnDisplayName() string {
	return u.Username
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}
