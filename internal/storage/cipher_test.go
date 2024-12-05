package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aexvir/skladka/internal/storage"
)

func TestCipher(t *testing.T) {
	key := "supersecretkey=="

	cipher := storage.NewCipher(key)
	encrypted, encerr := cipher.Encrypt("test")
	require.NoError(t, encerr)

	decrypted, decerr := cipher.Decrypt(encrypted)
	require.NoError(t, decerr)

	require.Equal(t, "test", decrypted)
}
