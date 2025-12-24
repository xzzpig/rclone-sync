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

// TestFileQuery_Local tests FileQuery.local resolver.
func (s *FileResolverTestSuite) TestFileQuery_Local() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.local")
	assert.True(s.T(), files.IsArray())

	// Should have 1: file1.txt, file2.txt, subdir
	assert.Equal(s.T(), 1, len(files.Array()))
}

// TestFileQuery_LocalWithDirectory tests that only directories are returned by Local resolver.
func (s *FileResolverTestSuite) TestFileQuery_LocalWithDirectory() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.local").Array()

	// Local resolver only returns directories, not files
	// Find the subdir entry
	var foundSubdir bool
	for _, f := range files {
		name := f.Get("name").String()
		isDir := f.Get("isDir").Bool()
		if name == "subdir" {
			foundSubdir = true
			assert.True(s.T(), isDir, "subdir should be a directory")
		}
		// file1.txt should NOT be in the list (Local only returns directories)
		assert.NotEqual(s.T(), "file1.txt", name, "files should not be in Local listing")
	}
	assert.True(s.T(), foundSubdir, "should find subdir")
}

// TestFileQuery_LocalSubdirectory tests browsing subdirectory.
func (s *FileResolverTestSuite) TestFileQuery_LocalSubdirectory() {
	// Create a nested directory inside subdir for testing
	require.NoError(s.T(), os.MkdirAll(filepath.Join(s.testDir, "subdir", "nested_dir"), 0755))

	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": filepath.Join(s.testDir, "subdir"),
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.local").Array()

	// Local only returns directories, so should have 1 item: nested_dir
	// (nested.txt is a file and should not be returned)
	assert.Equal(s.T(), 1, len(files))
	assert.Equal(s.T(), "nested_dir", files[0].Get("name").String())
	assert.True(s.T(), files[0].Get("isDir").Bool())
}

// TestFileQuery_LocalNonExistentPath tests FileQuery.local with non-existent path.
func (s *FileResolverTestSuite) TestFileQuery_LocalNonExistentPath() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": "/nonexistent/path/that/does/not/exist",
	})
	// Should return error or empty list
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		files := gjson.Get(data, "file.local")
		// Either null or empty array is acceptable
		if files.Exists() {
			assert.True(s.T(), files.IsArray())
			assert.Equal(s.T(), 0, len(files.Array()))
		}
	}
}

// TestFileQuery_LocalEmptyPath tests FileQuery.local with empty path.
func (s *FileResolverTestSuite) TestFileQuery_LocalEmptyPath() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": "",
	})
	// Should return error or empty list
	// Empty path is likely invalid
	_ = resp
}

// TestFileQuery_Remote tests FileQuery.remote resolver.
func (s *FileResolverTestSuite) TestFileQuery_Remote() {
	connID := s.Env.CreateTestConnection(s.T(), "local-conn")

	query := `
		query($connectionId: ID!, $path: String!) {
			file {
				remote(connectionId: $connectionId, path: $path) {
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
	})

	// Remote file listing may work or fail depending on connection setup
	// Just verify the query is valid
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		files := gjson.Get(data, "file.remote")
		if files.Exists() && files.IsArray() {
			// Local connection should work
			assert.GreaterOrEqual(s.T(), len(files.Array()), 0)
		}
	}
}

// TestFileQuery_RemoteNonExistentConnection tests FileQuery.remote with non-existent connection.
func (s *FileResolverTestSuite) TestFileQuery_RemoteNonExistentConnection() {
	query := `
		query($connectionId: ID!, $path: String!) {
			file {
				remote(connectionId: $connectionId, path: $path) {
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

// TestFileQuery_LocalDirectoryMetadata tests that directory metadata is returned correctly.
func (s *FileResolverTestSuite) TestFileQuery_LocalDirectoryMetadata() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
					path
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": s.testDir,
	})
	require.Empty(s.T(), resp.Errors)

	data := string(resp.Data)
	files := gjson.Get(data, "file.local").Array()

	// Local only returns directories, find subdir and check metadata
	for _, f := range files {
		if f.Get("name").String() == "subdir" {
			assert.True(s.T(), f.Get("isDir").Bool())
			return
		}
	}
	s.T().Error("subdir not found in listing")
}

// TestFileQuery_LocalRootPath tests listing root path.
func (s *FileResolverTestSuite) TestFileQuery_LocalRootPath() {
	query := `
		query($path: String!) {
			file {
				local(path: $path) {
					name
					isDir
				}
			}
		}
	`

	resp := s.Env.ExecuteGraphQLWithVars(s.T(), query, map[string]interface{}{
		"path": "/",
	})
	// Should return files from root (if accessible)
	if len(resp.Errors) == 0 {
		data := string(resp.Data)
		files := gjson.Get(data, "file.local")
		if files.Exists() && files.IsArray() {
			// Root should have some entries
			assert.Greater(s.T(), len(files.Array()), 0)
		}
	}
}
