// Package crypto provides encryption and decryption utilities for sensitive configuration data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"

	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

const errCiphertextTooShort = errs.ConstError("ciphertext too short")

// Encryptor handles encryption and decryption of configuration data using AES-256-GCM.
// If no key is provided (empty string), data is stored unencrypted (plain JSON).
type Encryptor struct {
	key       []byte
	plaintext bool // true if encryption is disabled (empty key)
}

// NewEncryptor creates a new Encryptor with the given key.
// - If key is empty: encryption is disabled, data is stored as plain JSON
// - If key is non-empty: it will be hashed using SHA-256 to produce a 32-byte key for AES-256
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		// Empty key means no encryption
		return &Encryptor{plaintext: true}, nil
	}

	// Use SHA-256 to derive a 32-byte key from any input length
	hash := sha256.Sum256([]byte(key))
	return &Encryptor{key: hash[:], plaintext: false}, nil
}

// EncryptConfig encrypts a configuration map into bytes.
// - If encryption is enabled (key provided): uses AES-256-GCM encryption
// - If encryption is disabled (no key): returns plain JSON
func (e *Encryptor) EncryptConfig(config map[string]string) ([]byte, error) {
	// Serialize config to JSON
	plaintext, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// If plaintext mode, return JSON directly
	if e.plaintext {
		return plaintext, nil
	}

	// Create AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt: nonce is prepended to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptConfig decrypts bytes back to a configuration map.
// - If encryption is enabled (key provided): decrypts using AES-256-GCM
// - If encryption is disabled (no key): parses as plain JSON
func (e *Encryptor) DecryptConfig(encrypted []byte) (map[string]string, error) {
	var plaintext []byte

	// If plaintext mode, data is already JSON
	if e.plaintext {
		plaintext = encrypted
	} else {
		// Create AES cipher
		block, err := aes.NewCipher(e.key)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}

		// Create GCM mode
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}

		// Check minimum length
		nonceSize := gcm.NonceSize()
		if len(encrypted) < nonceSize {
			return nil, errCiphertextTooShort
		}

		// Extract nonce and ciphertext
		nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

		// Decrypt
		plaintext, err = gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt: %w", err)
		}
	}

	// Deserialize JSON
	var config map[string]string
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
