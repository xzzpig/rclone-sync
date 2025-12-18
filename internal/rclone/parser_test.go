package rclone

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T064 [P] [US7] 单元测试：ParseRcloneConf() 解析
func TestParseRcloneConf(t *testing.T) {
	t.Run("parse valid config with single connection", func(t *testing.T) {
		content := `[test_local]
type = local
name = test_local
`
		connections, err := ParseRcloneConf(content)
		require.NoError(t, err)
		require.Len(t, connections, 1)

		conn := connections[0]
		assert.Equal(t, "test_local", conn.Name)
		assert.Equal(t, "local", conn.Type)
		assert.Equal(t, map[string]string{
			"type": "local",
			"name": "test_local",
		}, conn.Config)
	})

	t.Run("parse valid config with multiple connections", func(t *testing.T) {
		content := `[test_local]
type = local

[test_s3]
type = s3
access_key_id = AKIAIOSFODNN7EXAMPLE
secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1

[test_drive]
type = drive
client_id = 123456789.apps.googleusercontent.com
client_secret = secret123
`
		connections, err := ParseRcloneConf(content)
		require.NoError(t, err)
		require.Len(t, connections, 3)

		// Check first connection
		assert.Equal(t, "test_local", connections[0].Name)
		assert.Equal(t, "local", connections[0].Type)

		// Check second connection
		assert.Equal(t, "test_s3", connections[1].Name)
		assert.Equal(t, "s3", connections[1].Type)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", connections[1].Config["access_key_id"])
		assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", connections[1].Config["secret_access_key"])
		assert.Equal(t, "us-east-1", connections[1].Config["region"])

		// Check third connection
		assert.Equal(t, "test_drive", connections[2].Name)
		assert.Equal(t, "drive", connections[2].Type)
	})

	t.Run("parse config with OAuth token", func(t *testing.T) {
		content := `[test_onedrive]
type = onedrive
drive_id = A769229B43C2B2BD
drive_type = personal
token = {"access_token":"abc","token_type":"Bearer","refresh_token":"xyz","expiry":"2025-12-15T20:59:02.502619872+08:00"}
`
		connections, err := ParseRcloneConf(content)
		require.NoError(t, err)
		require.Len(t, connections, 1)

		conn := connections[0]
		assert.Equal(t, "test_onedrive", conn.Name)
		assert.Equal(t, "onedrive", conn.Type)
		assert.Contains(t, conn.Config["token"], "access_token")
	})

	t.Run("parse config with comments", func(t *testing.T) {
		content := `# This is a comment
[test_local]
type = local
# Another comment
name = test_local
`
		connections, err := ParseRcloneConf(content)
		require.NoError(t, err)
		require.Len(t, connections, 1)
		assert.Equal(t, "test_local", connections[0].Name)
	})
}

// T065 [P] [US7] 单元测试：解析空/无效内容
func TestParseRcloneConf_EmptyAndInvalid(t *testing.T) {
	t.Run("empty content returns empty list", func(t *testing.T) {
		connections, err := ParseRcloneConf("")
		require.NoError(t, err)
		assert.Empty(t, connections)
	})

	t.Run("whitespace only returns empty list", func(t *testing.T) {
		connections, err := ParseRcloneConf("   \n\n  \t  ")
		require.NoError(t, err)
		assert.Empty(t, connections)
	})

	t.Run("comments only returns empty list", func(t *testing.T) {
		content := `# Comment 1
# Comment 2
# Comment 3`
		connections, err := ParseRcloneConf(content)
		require.NoError(t, err)
		assert.Empty(t, connections)
	})

	t.Run("connection without type field returns error", func(t *testing.T) {
		content := `[test_local]
name = test_local
path = /tmp
`
		connections, err := ParseRcloneConf(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field 'type'")
		assert.Nil(t, connections)
	})

	t.Run("invalid INI format returns error", func(t *testing.T) {
		content := `[test_local
type = local
`
		_, err := ParseRcloneConf(content)
		require.Error(t, err)
	})
}

// T066 [P] [US7] 单元测试：检测内部名称重复
func TestValidateImport(t *testing.T) {
	t.Run("no duplicates in parsed connections", func(t *testing.T) {
		parsed := []ParsedConnection{
			{Name: "conn1", Type: "local", Config: map[string]string{"type": "local"}},
			{Name: "conn2", Type: "s3", Config: map[string]string{"type": "s3"}},
		}
		existing := []string{}

		result := ValidateImport(parsed, existing)
		require.NotNil(t, result)
		assert.Len(t, result.Valid, 2)
		assert.Empty(t, result.Conflicts)
		assert.Empty(t, result.InternalDuplicates)
	})

	t.Run("detect internal duplicates in parsed connections", func(t *testing.T) {
		parsed := []ParsedConnection{
			{Name: "conn1", Type: "local", Config: map[string]string{"type": "local"}},
			{Name: "conn1", Type: "s3", Config: map[string]string{"type": "s3"}},
			{Name: "conn2", Type: "drive", Config: map[string]string{"type": "drive"}},
		}
		existing := []string{}

		result := ValidateImport(parsed, existing)
		require.NotNil(t, result)
		assert.Len(t, result.Valid, 1) // Only conn2 is valid
		assert.Empty(t, result.Conflicts)
		assert.Len(t, result.InternalDuplicates, 1)
		assert.Contains(t, result.InternalDuplicates, "conn1")
	})

	t.Run("detect conflicts with existing connections", func(t *testing.T) {
		parsed := []ParsedConnection{
			{Name: "conn1", Type: "local", Config: map[string]string{"type": "local"}},
			{Name: "conn2", Type: "s3", Config: map[string]string{"type": "s3"}},
			{Name: "conn3", Type: "drive", Config: map[string]string{"type": "drive"}},
		}
		existing := []string{"conn1", "conn3"}

		result := ValidateImport(parsed, existing)
		require.NotNil(t, result)
		assert.Len(t, result.Valid, 1) // Only conn2 is valid
		assert.Len(t, result.Conflicts, 2)
		assert.Contains(t, result.Conflicts, "conn1")
		assert.Contains(t, result.Conflicts, "conn3")
		assert.Empty(t, result.InternalDuplicates)
	})

	t.Run("detect both internal duplicates and conflicts", func(t *testing.T) {
		parsed := []ParsedConnection{
			{Name: "conn1", Type: "local", Config: map[string]string{"type": "local"}},
			{Name: "conn1", Type: "s3", Config: map[string]string{"type": "s3"}},
			{Name: "conn2", Type: "drive", Config: map[string]string{"type": "drive"}},
		}
		existing := []string{"conn1", "conn2"}

		result := ValidateImport(parsed, existing)
		require.NotNil(t, result)
		assert.Empty(t, result.Valid) // No valid connections
		assert.Len(t, result.Conflicts, 1)
		assert.Contains(t, result.Conflicts, "conn2")
		assert.Len(t, result.InternalDuplicates, 1)
		assert.Contains(t, result.InternalDuplicates, "conn1")
	})

	t.Run("empty parsed connections", func(t *testing.T) {
		parsed := []ParsedConnection{}
		existing := []string{"conn1"}

		result := ValidateImport(parsed, existing)
		require.NotNil(t, result)
		assert.Empty(t, result.Valid)
		assert.Empty(t, result.Conflicts)
		assert.Empty(t, result.InternalDuplicates)
	})
}
