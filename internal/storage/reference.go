package storage

import (
	"crypto/rand"
	"strings"
)

const (
	refIdentifierLength = 8
	characters          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func generateReferenceIdentifier() (string, error) {
	buf := make([]byte, refIdentifierLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	var ref strings.Builder
	ref.Grow(refIdentifierLength)

	for i := range buf {
		ref.WriteByte(characters[int(buf[i])%len(characters)])
	}

	return ref.String(), nil
}
