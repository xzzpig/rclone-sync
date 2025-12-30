package rclone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestValidateFilterRules(t *testing.T) {
	t.Run("ValidRules", func(t *testing.T) {
		tests := []struct {
			name  string
			rules []string
		}{
			{
				name:  "Single exclude rule",
				rules: []string{"- node_modules/**"},
			},
			{
				name:  "Single include rule",
				rules: []string{"+ *.jpg"},
			},
			{
				name:  "Multiple rules",
				rules: []string{"- node_modules/**", "- .git/**", "- *.tmp", "+ **"},
			},
			{
				name:  "Include all",
				rules: []string{"+ **"},
			},
			{
				name:  "Exclude all",
				rules: []string{"- *"},
			},
			{
				name:  "Complex pattern with directory",
				rules: []string{"- /backup/**", "+ /documents/**", "- **"},
			},
			{
				name:  "Pattern with extension",
				rules: []string{"+ *.jpg", "+ *.png", "+ *.gif", "- *"},
			},
			{
				name:  "Pattern with question mark wildcard",
				rules: []string{"- file?.txt"},
			},
			{
				name:  "Pattern with bracket wildcard",
				rules: []string{"- file[0-9].txt"},
			},
			{
				name:  "Empty rules list",
				rules: []string{},
			},
			{
				name:  "Rule with unicode characters",
				rules: []string{"- 文档/**", "+ 图片/*.jpg"},
			},
			{
				name:  "Rule with spaces in pattern",
				rules: []string{"- My Documents/**"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := rclone.ValidateFilterRules(tt.rules)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("InvalidRules", func(t *testing.T) {
		tests := []struct {
			name        string
			rules       []string
			expectError bool
		}{
			{
				name:        "Missing prefix",
				rules:       []string{"node_modules/**"},
				expectError: true,
			},
			{
				name:        "Invalid prefix character",
				rules:       []string{"* node_modules/**"},
				expectError: true,
			},
			{
				name:        "Only prefix without space",
				rules:       []string{"-"},
				expectError: true,
			},
			{
				name:        "Mixed valid and invalid rules",
				rules:       []string{"- node_modules/**", "invalid_rule", "+ **"},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := rclone.ValidateFilterRules(tt.rules)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ErrorMessageContainsRuleInfo", func(t *testing.T) {
		rules := []string{"- valid/**", "invalid_rule", "+ **"}
		err := rclone.ValidateFilterRules(rules)
		require.Error(t, err)

		// Check the error is an I18nError
		i18nErr, ok := i18n.IsI18nError(err)
		require.True(t, ok, "Error should be an I18nError")

		// Verify the error data contains the correct information
		assert.Contains(t, i18nErr.Data, "Index")
		assert.Contains(t, i18nErr.Data, "Rule")
		assert.Contains(t, i18nErr.Data, "Reason")

		// Index should be 2 (1-based, so index 1 becomes 2)
		assert.Equal(t, 2, i18nErr.Data["Index"])
		// Rule should be the invalid rule
		assert.Equal(t, "invalid_rule", i18nErr.Data["Rule"])
		// Reason should contain the rclone error message
		assert.Contains(t, i18nErr.Data["Reason"].(string), "malformed rule")
	})
}

func TestValidateFilterRules_EdgeCases(t *testing.T) {
	t.Run("NilRules", func(t *testing.T) {
		err := rclone.ValidateFilterRules(nil)
		assert.NoError(t, err)
	})

	t.Run("RulesWithWhitespace", func(t *testing.T) {
		// Rules with leading/trailing whitespace in pattern
		rules := []string{"- file.txt "}
		err := rclone.ValidateFilterRules(rules)
		// This should be valid - rclone handles trailing spaces
		assert.NoError(t, err)
	})

	t.Run("SpecialCharactersInPattern", func(t *testing.T) {
		// Special characters that should be handled
		rules := []string{
			"- file (1).txt",
			"- file-name.txt",
			"- file_name.txt",
			"- file.name.txt",
		}
		err := rclone.ValidateFilterRules(rules)
		assert.NoError(t, err)
	})

	t.Run("OfficeTemporaryFiles", func(t *testing.T) {
		// Common temporary file patterns
		rules := []string{
			"- ~$*",
			"- *.tmp",
			"- *.bak",
			"- .DS_Store",
			"- Thumbs.db",
		}
		err := rclone.ValidateFilterRules(rules)
		assert.NoError(t, err)
	})

	t.Run("CommonExcludePatterns", func(t *testing.T) {
		// Common patterns users might use
		rules := []string{
			"- node_modules/**",
			"- .git/**",
			"- __pycache__/**",
			"- *.pyc",
			"- .venv/**",
			"- vendor/**",
			"- target/**",
			"- build/**",
			"- dist/**",
			"+ **",
		}
		err := rclone.ValidateFilterRules(rules)
		assert.NoError(t, err)
	})
}
