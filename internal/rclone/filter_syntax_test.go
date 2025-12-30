package rclone

import (
	"context"
	"testing"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

// TestRcloneFilterSyntaxSupport verifies that rclone's filter package supports
// the filter syntax patterns we rely on. This test serves as a compatibility
// check for the rclone version we're using as a dependency.
//
// If this test fails after upgrading rclone, it indicates that the filter syntax
// behavior has changed and our code needs to be updated accordingly.
func TestRcloneFilterSyntaxSupport(t *testing.T) {
	t.Run("BasicPatterns", func(t *testing.T) {
		tests := []struct {
			name     string
			rule     string
			testPath string
			isDir    bool
			expect   bool // true = included, false = excluded
		}{
			// Exclude patterns
			{
				name:     "Exclude single file by name",
				rule:     "- test.txt",
				testPath: "test.txt",
				isDir:    false,
				expect:   false,
			},
			{
				name:     "Exclude by extension",
				rule:     "- *.tmp",
				testPath: "file.tmp",
				isDir:    false,
				expect:   false,
			},
			{
				name:     "Exclude directory recursive",
				rule:     "- node_modules/**",
				testPath: "node_modules/package/index.js",
				isDir:    false,
				expect:   false,
			},
			{
				name:     "Exclude hidden files",
				rule:     "- .*",
				testPath: ".gitignore",
				isDir:    false,
				expect:   false,
			},
			// Include patterns
			{
				name:     "Include by extension",
				rule:     "+ *.jpg",
				testPath: "photo.jpg",
				isDir:    false,
				expect:   true,
			},
			{
				name:     "Include all recursively",
				rule:     "+ **",
				testPath: "any/path/file.txt",
				isDir:    false,
				expect:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fi, err := filter.NewFilter(nil)
				require.NoError(t, err)

				err = fi.AddRule(tt.rule)
				require.NoError(t, err)

				// Test the filter
				result := fi.Include(tt.testPath, 100, time.Now(), nil)
				assert.Equal(t, tt.expect, result,
					"Filter rule %q applied to %q should return %v", tt.rule, tt.testPath, tt.expect)
			})
		}
	})

	t.Run("MultipleRulesFirstMatch", func(t *testing.T) {
		// Test that first matching rule wins
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)

		// Add rules in order: exclude .tmp, include all
		require.NoError(t, fi.AddRule("- *.tmp"))
		require.NoError(t, fi.AddRule("+ **"))

		// .tmp files should be excluded (first rule matches)
		assert.False(t, fi.Include("file.tmp", 100, time.Now(), nil),
			"*.tmp files should be excluded by first rule")

		// Other files should be included (second rule matches)
		assert.True(t, fi.Include("file.txt", 100, time.Now(), nil),
			"*.txt files should be included by second rule")
	})

	t.Run("DirectoryPatterns", func(t *testing.T) {
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)

		// Exclude node_modules directory and all contents
		require.NoError(t, fi.AddRule("- node_modules/**"))
		require.NoError(t, fi.AddRule("+ **"))

		// Files inside node_modules should be excluded
		assert.False(t, fi.Include("node_modules/lodash/index.js", 100, time.Now(), nil))
		assert.False(t, fi.Include("node_modules/react/package.json", 100, time.Now(), nil))

		// Files outside node_modules should be included
		assert.True(t, fi.Include("src/index.js", 100, time.Now(), nil))
		assert.True(t, fi.Include("package.json", 100, time.Now(), nil))
	})

	t.Run("WildcardPatterns", func(t *testing.T) {
		tests := []struct {
			name     string
			rule     string
			path     string
			expected bool
		}{
			{
				name:     "Single char wildcard ?",
				rule:     "- file?.txt",
				path:     "file1.txt",
				expected: false,
			},
			{
				name:     "Single char wildcard no match",
				rule:     "- file?.txt",
				path:     "file10.txt",
				expected: true, // ? only matches single char
			},
			{
				name:     "Star matches any in single level",
				rule:     "- *.log",
				path:     "app.log",
				expected: false,
			},
			{
				name:     "Double star matches any depth",
				rule:     "- logs/**",
				path:     "logs/2024/01/app.log",
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fi, err := filter.NewFilter(nil)
				require.NoError(t, err)
				require.NoError(t, fi.AddRule(tt.rule))
				require.NoError(t, fi.AddRule("+ **"))

				result := fi.Include(tt.path, 100, time.Now(), nil)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("UnicodePatterns", func(t *testing.T) {
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)

		// Unicode characters in patterns
		require.NoError(t, fi.AddRule("- 文档/**"))
		require.NoError(t, fi.AddRule("+ **"))

		// Chinese directory should be excluded
		assert.False(t, fi.Include("文档/报告.docx", 100, time.Now(), nil))

		// Other paths should be included
		assert.True(t, fi.Include("documents/report.docx", 100, time.Now(), nil))
	})

	t.Run("SpacesInPatterns", func(t *testing.T) {
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)

		// Patterns with spaces
		require.NoError(t, fi.AddRule("- My Documents/**"))
		require.NoError(t, fi.AddRule("+ **"))

		assert.False(t, fi.Include("My Documents/file.txt", 100, time.Now(), nil))
		assert.True(t, fi.Include("MyDocuments/file.txt", 100, time.Now(), nil))
	})

	t.Run("CommonExcludePatterns", func(t *testing.T) {
		// Test common patterns that users will likely use
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)

		commonExcludes := []string{
			"- node_modules/**",
			"- .git/**",
			"- __pycache__/**",
			"- *.pyc",
			"- .DS_Store",
			"- Thumbs.db",
			"- ~$*",
			"- *.tmp",
			"- .venv/**",
			"+ **",
		}

		for _, rule := range commonExcludes {
			require.NoError(t, fi.AddRule(rule), "Rule should be valid: %s", rule)
		}

		// Test exclusions
		assert.False(t, fi.Include("node_modules/lodash/index.js", 100, time.Now(), nil))
		assert.False(t, fi.Include(".git/config", 100, time.Now(), nil))
		assert.False(t, fi.Include("__pycache__/module.pyc", 100, time.Now(), nil))
		assert.False(t, fi.Include("script.pyc", 100, time.Now(), nil))
		assert.False(t, fi.Include(".DS_Store", 100, time.Now(), nil))
		assert.False(t, fi.Include("Thumbs.db", 100, time.Now(), nil))
		assert.False(t, fi.Include("~$Document.docx", 100, time.Now(), nil))
		assert.False(t, fi.Include("temp.tmp", 100, time.Now(), nil))
		assert.False(t, fi.Include(".venv/lib/python3.11/site-packages/pip/main.py", 100, time.Now(), nil))

		// Test inclusions
		assert.True(t, fi.Include("src/main.py", 100, time.Now(), nil))
		assert.True(t, fi.Include("README.md", 100, time.Now(), nil))
		assert.True(t, fi.Include("package.json", 100, time.Now(), nil))
	})
}

// TestFilterContextIntegration tests that filters can be properly integrated
// with the rclone context, which is how we'll use them in sync operations.
func TestFilterContextIntegration(t *testing.T) {
	t.Run("FilterReplaceConfig", func(t *testing.T) {
		ctx := context.Background()

		// Create filter with some rules
		fi, err := filter.NewFilter(nil)
		require.NoError(t, err)
		require.NoError(t, fi.AddRule("- *.tmp"))
		require.NoError(t, fi.AddRule("+ **"))

		// Inject filter into context
		ctx = filter.ReplaceConfig(ctx, fi)

		// Retrieve filter from context
		retrievedFilter := filter.GetConfig(ctx)
		require.NotNil(t, retrievedFilter)

		// Verify filter works as expected
		assert.False(t, retrievedFilter.Include("file.tmp", 100, time.Now(), nil))
		assert.True(t, retrievedFilter.Include("file.txt", 100, time.Now(), nil))
	})
}

// TestFsConfigIntegration tests that fs.AddConfig works for setting transfers
func TestFsConfigIntegration(t *testing.T) {
	t.Run("SetTransfers", func(t *testing.T) {
		ctx := context.Background()

		// Get original transfers value
		originalCi := fs.GetConfig(ctx)
		originalTransfers := originalCi.Transfers

		// Use AddConfig to get a new context with modifiable config
		ctx, ci := fs.AddConfig(ctx)
		ci.Transfers = 8

		// Verify config was updated in the new context
		newCi := fs.GetConfig(ctx)
		assert.Equal(t, 8, newCi.Transfers)

		// Original context should be unchanged
		originalCi2 := fs.GetConfig(context.Background())
		assert.Equal(t, originalTransfers, originalCi2.Transfers)
	})
}

// TestCreateFilterFromRules tests the createFilterFromRules helper function.
// This function is an unexported helper used by ValidateFilterRules, applyFilterRules,
// and ListRemoteDir to eliminate code duplication.
func TestCreateFilterFromRules(t *testing.T) {
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
				name:  "Complex patterns",
				rules: []string{"+ *.go", "+ *.md", "- *"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fi, err := createFilterFromRules(tt.rules)
				require.NoError(t, err)
				assert.NotNil(t, fi)

				// Verify the filter was created correctly by testing with a sample path
				if len(tt.rules) > 0 {
					// Test that we can use the filter to check paths
					// This verifies the filter was properly initialized
					_ = fi.Include("test.txt", 100, time.Now(), nil)
				}
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
				name:        "Mixed valid and invalid rules - second rule invalid",
				rules:       []string{"- node_modules/**", "invalid_rule"},
				expectError: true,
			},
			{
				name:        "Mixed valid and invalid rules - first rule invalid",
				rules:       []string{"invalid_rule", "+ **"},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fi, err := createFilterFromRules(tt.rules)
				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, fi)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, fi)
				}
			})
		}
	})

	t.Run("ErrorMessageContainsRuleInfo", func(t *testing.T) {
		rules := []string{"- valid/**", "invalid_rule", "+ **"}
		fi, err := createFilterFromRules(rules)
		require.Error(t, err)
		require.Nil(t, fi)

		// Check the error is an I18nError
		i18nErr, ok := i18n.IsI18nError(err)
		require.True(t, ok, "Error should be an I18nError")

		// Verify the error data contains the correct information
		assert.Contains(t, i18nErr.Data, "Index")
		assert.Contains(t, i18nErr.Data, "Rule")
		assert.Contains(t, i18nErr.Data, "Reason")
	})

	t.Run("FilterBehavior", func(t *testing.T) {
		t.Run("ExcludePattern", func(t *testing.T) {
			fi, err := createFilterFromRules([]string{"- *.tmp"})
			require.NoError(t, err)

			// Test that *.tmp files are excluded
			result := fi.Include("file.tmp", 100, time.Now(), nil)
			assert.False(t, result, "*.tmp files should be excluded")

			// Test that other files are included (default behavior)
			result = fi.Include("file.txt", 100, time.Now(), nil)
			assert.True(t, result, ".txt files should be included by default")
		})

		t.Run("IncludePattern", func(t *testing.T) {
			fi, err := createFilterFromRules([]string{"+ *.jpg", "- *"})
			require.NoError(t, err)

			// Test that *.jpg files are included
			result := fi.Include("photo.jpg", 1024, time.Now(), nil)
			assert.True(t, result, "*.jpg files should be included")

			// Test that other files are excluded
			result = fi.Include("file.txt", 100, time.Now(), nil)
			assert.False(t, result, ".txt files should be excluded by - *")
		})

		t.Run("DirectoryPattern", func(t *testing.T) {
			fi, err := createFilterFromRules([]string{"- node_modules/**", "+ **"})
			require.NoError(t, err)

			// Test that node_modules directory is excluded
			result := fi.Include("node_modules/package/index.js", 0, time.Now(), nil)
			assert.False(t, result, "node_modules/** should exclude the directory")

			// Test that other directories are included
			result = fi.Include("src/main.go", 1024, time.Now(), nil)
			assert.True(t, result, "src/main.go should be included")
		})
	})
}
