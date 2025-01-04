package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"io"

	"golang.org/x/crypto/argon2"
)

type Cipher struct {
	key []byte
}

func NewCipher(key, salt string) *Cipher {
	return &Cipher{
		key: argon2.IDKey([]byte(key), []byte(salt), 3, 64*1024, 2, 32),
	}
}

func (c *Cipher) Hash(value string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		panic(err)
	}

	hash := argon2.IDKey([]byte(value), salt, 3, 64*1024, 2, 32)

	return base64.URLEncoding.EncodeToString(
		append(salt, hash...),
	)
}

func (c *Cipher) Verify(password, encoded string) bool {
	combined, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		panic(err)
	}

	salt, storedhash := combined[:16], combined[16:]

	computedhash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)

	return subtle.ConstantTimeCompare(storedhash, computedhash) == 1
}

func (c *Cipher) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (c *Cipher) Decrypt(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	noncesz := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:noncesz], ciphertext[noncesz:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
