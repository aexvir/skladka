package auth

import (
	"time"
)

// Session represents a WebAuthn user session.
type Session struct {
	Username  string
	Token     string
	Data      []byte
	ExpiresAt time.Time
	Throwaway bool
}

// IsExpired returns true if the session is expired.
func (s Session) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now())
}
