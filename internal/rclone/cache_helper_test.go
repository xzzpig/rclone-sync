package rclone

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T048: 单元测试：IsConnectionLoaded() 缓存检查
func TestIsConnectionLoaded(t *testing.T) {
	t.Run("returns false for never-loaded connection", func(t *testing.T) {
		// A connection that was never accessed should return false
		result := IsConnectionLoaded("never-loaded-remote")
		assert.False(t, result)
	})

	t.Run("returns false for non-existing connection", func(t *testing.T) {
		result := IsConnectionLoaded("non-existing-remote-12345")
		assert.False(t, result)
	})

	t.Run("returns false for empty name", func(t *testing.T) {
		result := IsConnectionLoaded("")
		assert.False(t, result)
	})

	// Note: Testing "loaded" state would require actually loading a connection
	// through rclone's fs.NewFs, which needs a real remote configuration.
	// This is better tested in integration tests.
}
