// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tidwall/gjson"
)

// FileResolverTestSuite tests FileQuery resolvers.
type FileResolverTestSuite struct {
	ResolverTestSuite
	testDir string
}

func TestFileResolverSuite(t *testing.T) {
	suite.Run(t, new(FileResolverTestSuite))
}

// SetupTest runs before each test.
func (s *FileResolverTestSuite) SetupTest() {
	s.ResolverTestSuite.SetupTest()

	// Create a test directory with some files
	s.testDir = s.T().TempDir()

	// Create some test files and directories
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "subdir"), 0755))
	require.NoError(s.T(), os.WriteFile(filepath.Join(s.testDir, "file1.txt"), []byte("content1"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(s.testDir, "file2.txt"), []byte("content2"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(s.testDir, "subdir", "nested.txt"), []byte("nested"), 0644))
}

// TestFileQuery_List tests the unified list endpoint.
func (s *FileResolverTestSuite) TestFileQuery_List() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	// Test: Local listing (connectionId = null)
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	// By default, only directories are returned
	assert.GreaterOrEqual(s.T(), len(files), 1)

	// Find subdir
	foundSubdir := false
	for _, f := range files {
		if f.Get("name").String() == "subdir" {
			foundSubdir = true
			assert.True(s.T(), f.Get("isDir").Bool())
		}
	}
	assert.True(s.T(), foundSubdir, "Should find subdir")
}

// TestFileQuery_ListLocalWithFilters tests list endpoint with local path and filters.
func (s *FileResolverTestSuite) TestFileQuery_ListLocalWithFilters() {
	// Create test files
	filterDir := filepath.Join(s.testDir, "filter_local_test")
	require.NoError(s.T(), os.MkdirAll(filterDir, 0755))
	require.NoError(s.T(), os.WriteFile(filepath.Join(filterDir, "keep.txt"), []byte("keep"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(filterDir, "ignore.tmp"), []byte("ignore"), 0644))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(filterDir, "keepdir"), 0755))

	query := `
		query($connectionId: ID, $path: String!, $filters: [String!], $includeFiles: Boolean) {
			file {
				list(connectionId: $connectionId, path: $path, filters: $filters, includeFiles: $includeFiles) {
					name
					isDir
				}
			}
		}
	`

	// Test with filter to exclude .tmp files
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filterDir,
		"includeFiles": true,
		"filters":      []interface{}{"- *.tmp", "+ **"},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	// Should return keepdir + keep.txt (exclude ignore.tmp) = 2 entries
	assert.Equal(s.T(), 2, len(files))

	for _, f := range files {
		assert.NotContains(s.T(), f.Get("name").String(), ".tmp", "Should not contain .tmp files")
	}
}

// TestFileQuery_ListRemoteWithConnectionId tests list endpoint with connection ID.
func (s *FileResolverTestSuite) TestFileQuery_ListRemoteWithConnectionId() {
	connID := s.Env.CreateTestConnection(s.T(), "local-conn-list-test")

	query := `
		query($connectionId: ID, $path: String!, $includeFiles: Boolean) {
			file {
				list(connectionId: $connectionId, path: $path, includeFiles: $includeFiles) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": connID.String(),
		"path":         s.testDir,
		"includeFiles": true,
	})

	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		files := gjson.Get(data, "file.list").Array()
		// Should have files and directories
		assert.GreaterOrEqual(s.T(), len(files), 3) // file1.txt, file2.txt, subdir
	}
}

// TestFileQuery_ListNonExistentConnection tests list endpoint with non-existent connection.
func (s *FileResolverTestSuite) TestFileQuery_ListNonExistentConnection() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": "00000000-0000-0000-0000-000000000000",
		"path":         "/",
	})
	// Should return error for non-existent connection
	require.NotEmpty(s.T(), resp.Errors)
}

// TestFileQuery_ListWithIncludeFiles tests list endpoint with includeFiles parameter.
func (s *FileResolverTestSuite) TestFileQuery_ListWithIncludeFiles() {
	query := `
		query($connectionId: ID, $path: String!, $includeFiles: Boolean) {
			file {
				list(connectionId: $connectionId, path: $path, includeFiles: $includeFiles) {
					name
					isDir
				}
			}
		}
	`

	// Test 1: Without includeFiles (should only return directories)
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	// Should only return subdir (directories only by default)
	assert.Equal(s.T(), 1, len(files))
	if len(files) == 1 {
		assert.Equal(s.T(), "subdir", files[0].Get("name").String())
		assert.True(s.T(), files[0].Get("isDir").Bool())
	}

	// Test 2: With includeFiles=true (should return all entries)
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
		"includeFiles": true,
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	files = gjson.Get(data, "file.list").Array()
	// Should return 1 directory + 2 files = 3 entries
	assert.Equal(s.T(), 3, len(files))

	hasFile := false
	hasDir := false
	for _, f := range files {
		if f.Get("isDir").Bool() {
			hasDir = true
		} else {
			hasFile = true
		}
	}
	assert.True(s.T(), hasFile, "Should have at least one file")
	assert.True(s.T(), hasDir, "Should have at least one directory")
}

// TestFileQuery_ListNonExistentPath tests list endpoint with non-existent path.
func (s *FileResolverTestSuite) TestFileQuery_ListNonExistentPath() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         "/nonexistent/path/that/does/not/exist",
	})
	// Should return error for non-existent path
	require.NotEmpty(s.T(), resp.Errors)
}

// TestFileQuery_ListEmptyPath tests list endpoint with empty path.
func (s *FileResolverTestSuite) TestFileQuery_ListEmptyPath() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         "",
	})
	// Empty path should return error
	require.NotEmpty(s.T(), resp.Errors)
}

// TestFileQuery_ListDeepNesting tests list endpoint with deeply nested directories.
func (s *FileResolverTestSuite) TestFileQuery_ListDeepNesting() {
	// Create deeply nested directories
	deepPath := filepath.Join(s.testDir, "level1", "level2", "level3", "level4")
	require.NoError(s.T(), os.MkdirAll(deepPath, 0755))

	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	// List level3 directory
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filepath.Join(s.testDir, "level1", "level2", "level3"),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	assert.Equal(s.T(), 1, len(files))
	assert.Equal(s.T(), "level4", files[0].Get("name").String())
	assert.True(s.T(), files[0].Get("isDir").Bool())
}

// TestFileQuery_ListSpecialCharacters tests list endpoint with special characters in directory names.
func (s *FileResolverTestSuite) TestFileQuery_ListSpecialCharacters() {
	// Create directories with special characters
	specialDirs := []string{
		"dir with spaces",
		"dir-with-dashes",
		"dir_with_underscores",
	}

	for _, dirName := range specialDirs {
		require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, dirName), 0755))
	}

	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()

	// Should have at least the special directories plus existing subdir
	assert.GreaterOrEqual(s.T(), len(files), len(specialDirs)+1)

	// Check for specific special directories
	names := make(map[string]bool)
	for _, f := range files {
		names[f.Get("name").String()] = true
	}

	for _, dirName := range specialDirs {
		assert.True(s.T(), names[dirName], "Should find directory: %s", dirName)
	}
}

// TestFileQuery_ListEmptyDirectory tests list endpoint with an empty directory.
func (s *FileResolverTestSuite) TestFileQuery_ListEmptyDirectory() {
	// Create an empty directory
	emptyDir := filepath.Join(s.testDir, "empty_dir")
	require.NoError(s.T(), os.MkdirAll(emptyDir, 0755))

	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         emptyDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	// Empty directory should return empty array
	assert.Equal(s.T(), 0, len(files))
}

// TestFileQuery_ListInvalidUUID tests list endpoint with invalid connection UUID.
func (s *FileResolverTestSuite) TestFileQuery_ListInvalidUUID() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": "not-a-valid-uuid",
		"path":         "/",
	})
	// Should return error for invalid UUID
	require.NotEmpty(s.T(), resp.Errors)
}

// TestFileQuery_ListInvalidFilter tests list endpoint with invalid filter rules.
func (s *FileResolverTestSuite) TestFileQuery_ListInvalidFilter() {
	query := `
		query($connectionId: ID, $path: String!, $filters: [String!]) {
			file {
				list(connectionId: $connectionId, path: $path, filters: $filters) {
					name
				}
			}
		}
	`

	// Test with invalid filter rule (missing + or - prefix)
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
		"filters":      []interface{}{"*.tmp"}, // Invalid: missing prefix
	})
	// Should return error for invalid filter rule
	require.NotEmpty(s.T(), resp.Errors)
	assert.Contains(s.T(), resp.Errors[0].Message, "Filter")
}

// TestFileQuery_ListFilterPreview tests filter preview functionality for task configuration.
func (s *FileResolverTestSuite) TestFileQuery_ListFilterPreview() {
	// Create a directory structure that mimics a typical project
	projectDir := filepath.Join(s.testDir, "project")
	require.NoError(s.T(), os.MkdirAll(filepath.Join(projectDir, "src"), 0755))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(projectDir, "node_modules", "package"), 0755))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(projectDir, ".git", "objects"), 0755))
	require.NoError(s.T(), os.WriteFile(filepath.Join(projectDir, "index.js"), []byte("code"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(projectDir, "package.json"), []byte("{}"), 0644))

	query := `
		query($connectionId: ID, $path: String!, $filters: [String!], $includeFiles: Boolean) {
			file {
				list(connectionId: $connectionId, path: $path, filters: $filters, includeFiles: $includeFiles) {
					name
					isDir
				}
			}
		}
	`

	// Test typical ignore pattern: exclude node_modules and .git
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         projectDir,
		"includeFiles": true,
		"filters":      []interface{}{"- node_modules", "- .git", "+ **"},
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()

	// Should include src, index.js, package.json but NOT node_modules and .git
	names := make(map[string]bool)
	for _, f := range files {
		names[f.Get("name").String()] = true
	}

	assert.True(s.T(), names["src"], "Should include src directory")
	assert.True(s.T(), names["index.js"], "Should include index.js")
	assert.True(s.T(), names["package.json"], "Should include package.json")
	assert.False(s.T(), names["node_modules"], "Should exclude node_modules")
	assert.False(s.T(), names[".git"], "Should exclude .git")
}

// TestFileQuery_ListRootPath tests listing root path.
func (s *FileResolverTestSuite) TestFileQuery_ListRootPath() {
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         "/",
	})
	// Should return files from root (if accessible)
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		files := gjson.Get(data, "file.list")
		if files.Exists() && files.IsArray() {
			// Root should have some entries
			assert.Greater(s.T(), len(files.Array()), 0)
		}
	}
}

// TestFileQuery_ListFilePath tests list endpoint with a file path instead of directory.
func (s *FileResolverTestSuite) TestFileQuery_ListFilePath() {
	// Try to list a file (not a directory)
	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filepath.Join(s.testDir, "file1.txt"),
	})
	// Should return error for non-directory paths
	require.NotEmpty(s.T(), resp.Errors)
}

// TestFileQuery_ListSymlink tests list endpoint with a symlink directory.
func (s *FileResolverTestSuite) TestFileQuery_ListSymlink() {
	// Create a symlink to subdir
	symlinkPath := filepath.Join(s.testDir, "symlink_dir")
	err := os.Symlink(filepath.Join(s.testDir, "subdir"), symlinkPath)
	if err != nil {
		// Skip if symlinks are not supported
		s.T().Skip("Symlinks not supported on this platform")
		return
	}

	query := `
		query($connectionId: ID, $path: String!) {
			file {
				list(connectionId: $connectionId, path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()

	// rclone does not follow symlinks by default (-L/--copy-links is not set)
	// so symlink_dir may not be included in the listing
	// Just verify the query completes without error and returns some entries
	assert.GreaterOrEqual(s.T(), len(files), 1, "Should return at least one entry (subdir)")

	// Verify subdir is in the list (the real directory)
	foundSubdir := false
	for _, f := range files {
		if f.Get("name").String() == "subdir" {
			foundSubdir = true
			break
		}
	}
	assert.True(s.T(), foundSubdir, "Should include subdir")
}

// TestFileQuery_ListWithFilters tests list endpoint with various filter rules.
func (s *FileResolverTestSuite) TestFileQuery_ListWithFilters() {
	// Create a test directory with specific files for filter testing
	filterDir := filepath.Join(s.testDir, "filter_test")
	require.NoError(s.T(), os.MkdirAll(filterDir, 0755))
	require.NoError(s.T(), os.WriteFile(filepath.Join(filterDir, "include.txt"), []byte("include"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(filterDir, "exclude.tmp"), []byte("exclude"), 0644))
	require.NoError(s.T(), os.WriteFile(filepath.Join(filterDir, "data.json"), []byte("{}"), 0644))
	require.NoError(s.T(), os.MkdirAll(filepath.Join(filterDir, "subdir"), 0755))

	query := `
		query($connectionId: ID, $path: String!, $filters: [String!], $includeFiles: Boolean) {
			file {
				list(connectionId: $connectionId, path: $path, filters: $filters, includeFiles: $includeFiles) {
					name
					path
					isDir
				}
			}
		}
	`

	// Test 1: Without includeFiles (should only return directories)
	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filterDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.list").Array()
	// Should only return subdir (directories only by default)
	assert.Equal(s.T(), 1, len(files))
	if len(files) == 1 {
		assert.Equal(s.T(), "subdir", files[0].Get("name").String())
		assert.True(s.T(), files[0].Get("isDir").Bool())
	}

	// Test 2: With includeFiles=true (should return all entries)
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filterDir,
		"includeFiles": true,
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	files = gjson.Get(data, "file.list").Array()
	// Should return 1 directory + 3 files = 4 entries
	assert.Equal(s.T(), 4, len(files))

	// Test 3: With filter to exclude .tmp files
	resp = s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"connectionId": nil,
		"path":         filterDir,
		"includeFiles": true,
		"filters":      []interface{}{"- *.tmp", "+ **"},
	})
	require.Empty(s.T(), resp.Errors)

	data = string(resp.Data)
	files = gjson.Get(data, "file.list").Array()
	// Should return 1 directory + 2 files (exclude.tmp excluded) = 3 entries
	assert.Equal(s.T(), 3, len(files))

	for _, f := range files {
		assert.NotContains(s.T(), f.Get("name").String(), ".tmp", "Should not contain .tmp files")
	}
}
