package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

const nonceSize = 12

// Encrypt encrypts plaintext with AES-256-GCM.
// recordID is passed as AAD (prevents ciphertext-swap attacks).
// Output: 12-byte nonce || ciphertext (with GCM tag appended by Seal).
func Encrypt(plaintext, recordID, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("store: encrypt: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("store: encrypt: new gcm: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("store: encrypt: generate nonce: %w", err)
	}

	// Seal appends ciphertext+tag to nonce, so output is nonce || ciphertext || tag
	ciphertext := gcm.Seal(nonce, nonce, plaintext, recordID)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
// recordID must match what was used at encrypt time.
func Decrypt(ciphertext, recordID, key []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, errors.New("store: decrypt: ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("store: decrypt: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("store: decrypt: new gcm: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	ct := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ct, recordID)
	if err != nil {
		return nil, fmt.Errorf("store: decrypt: open: %w", err)
	}
	return plaintext, nil
}

// KeyFromEnv reads BRIDGE_ENCRYPTION_KEY env var (32-byte hex), returns the key bytes.
// Returns an error if the var is absent, not valid hex, or not 32 bytes.
func KeyFromEnv() ([]byte, error) {
	val := os.Getenv("BRIDGE_ENCRYPTION_KEY")
	if val == "" {
		return nil, errors.New("store: BRIDGE_ENCRYPTION_KEY env var not set")
	}
	key, err := hex.DecodeString(val)
	if err != nil {
		return nil, fmt.Errorf("store: BRIDGE_ENCRYPTION_KEY is not valid hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("store: BRIDGE_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}
	return key, nil
}
