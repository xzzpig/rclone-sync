package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncryptor(t *testing.T) {
	t.Run("valid key - any length", func(t *testing.T) {
		key := "my-secret-key"
		enc, err := NewEncryptor(key)
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("valid key - long passphrase", func(t *testing.T) {
		key := "this is a very long passphrase that users might prefer"
		enc, err := NewEncryptor(key)
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("valid key - short", func(t *testing.T) {
		key := "short"
		enc, err := NewEncryptor(key)
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("empty key - plaintext mode", func(t *testing.T) {
		enc, err := NewEncryptor("")
		require.NoError(t, err)
		assert.NotNil(t, enc)

		// Verify it works in plaintext mode
		config := map[string]string{"key": "value"}
		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		// In plaintext mode, output should be readable JSON
		assert.Contains(t, string(encrypted), "key")
		assert.Contains(t, string(encrypted), "value")

		// Should decrypt successfully
		decrypted, err := enc.DecryptConfig(encrypted)
		require.NoError(t, err)
		assert.Equal(t, config, decrypted)
	})
}

func TestEncryptDecryptConfig(t *testing.T) {
	key := "12345678901234567890123456789012"
	enc, err := NewEncryptor(key)
	require.NoError(t, err)

	t.Run("encrypt and decrypt simple config", func(t *testing.T) {
		config := map[string]string{
			"type":     "onedrive",
			"token":    "secret_token_value",
			"drive_id": "abc123",
		}

		// Encrypt
		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		// Verify encrypted data doesn't contain plain text
		assert.NotContains(t, string(encrypted), "secret_token_value")
		assert.NotContains(t, string(encrypted), "onedrive")

		// Decrypt
		decrypted, err := enc.DecryptConfig(encrypted)
		require.NoError(t, err)
		assert.Equal(t, config, decrypted)
	})

	t.Run("encrypt empty config", func(t *testing.T) {
		config := map[string]string{}
		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		decrypted, err := enc.DecryptConfig(encrypted)
		require.NoError(t, err)
		assert.Equal(t, config, decrypted)
	})

	t.Run("encrypt config with special characters", func(t *testing.T) {
		config := map[string]string{
			"password": "p@ssw0rd!#$%^&*()",
			"token":    `{"access":"abc","refresh":"xyz"}`,
		}

		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		decrypted, err := enc.DecryptConfig(encrypted)
		require.NoError(t, err)
		assert.Equal(t, config, decrypted)
	})

	t.Run("decrypt with wrong key fails", func(t *testing.T) {
		config := map[string]string{"key": "value"}

		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		// Create encryptor with different key
		wrongKey := "00000000000000000000000000000000"
		wrongEnc, err := NewEncryptor(wrongKey)
		require.NoError(t, err)

		// Decrypt should fail
		_, err = wrongEnc.DecryptConfig(encrypted)
		assert.Error(t, err)
	})

	t.Run("decrypt invalid data fails", func(t *testing.T) {
		invalidData := []byte("this is not encrypted data")
		_, err := enc.DecryptConfig(invalidData)
		assert.Error(t, err)
	})

	t.Run("decrypt truncated data fails", func(t *testing.T) {
		config := map[string]string{"key": "value"}
		encrypted, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		// Truncate the encrypted data
		truncated := encrypted[:len(encrypted)/2]
		_, err = enc.DecryptConfig(truncated)
		assert.Error(t, err)
	})

	t.Run("encrypt produces different ciphertext each time", func(t *testing.T) {
		config := map[string]string{"key": "value"}

		encrypted1, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		encrypted2, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		// Due to random nonce, ciphertexts should be different
		assert.NotEqual(t, encrypted1, encrypted2)

		// But both should decrypt to same value
		decrypted1, err := enc.DecryptConfig(encrypted1)
		require.NoError(t, err)
		decrypted2, err := enc.DecryptConfig(encrypted2)
		require.NoError(t, err)
		assert.Equal(t, decrypted1, decrypted2)
	})

	t.Run("plaintext mode with empty key", func(t *testing.T) {
		// Create encryptor with empty key (plaintext mode)
		enc, err := NewEncryptor("")
		require.NoError(t, err)

		config := map[string]string{
			"type":     "onedrive",
			"token":    "secret_token_value",
			"drive_id": "abc123",
		}

		// Store config (should be plain JSON)
		stored, err := enc.EncryptConfig(config)
		require.NoError(t, err)

		// Verify it's plain JSON (readable)
		assert.Contains(t, string(stored), "onedrive")
		assert.Contains(t, string(stored), "secret_token_value")
		assert.Contains(t, string(stored), "abc123")

		// Should retrieve successfully
		retrieved, err := enc.DecryptConfig(stored)
		require.NoError(t, err)
		assert.Equal(t, config, retrieved)
	})
}
