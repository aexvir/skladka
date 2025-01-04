package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/aexvir/skladka/internal/storage"
)

func TestCipherEncryption(t *testing.T) {
	key := "supersecretkey=="
	salt := "6370b25f61f2025a0d4fcbb4aaf8859f"

	cipher := storage.NewCipher(key, salt)
	encrypted, encerr := cipher.Encrypt("test")
	require.NoError(t, encerr)

	decrypted, decerr := cipher.Decrypt(encrypted)
	require.NoError(t, decerr)

	require.Equal(t, "test", decrypted)
}

func TestCipherHashing(t *testing.T) {
	key := "supersecretkey=="
	salt := "6370b25f61f2025a0d4fcbb4aaf8859f"

	cipher := storage.NewCipher(key, salt)

	password := "muchosecreto"
	encoded := cipher.Hash(password)

	require.True(t, cipher.Verify("muchosecreto", encoded))
}
