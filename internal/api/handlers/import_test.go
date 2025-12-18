package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent/enttest"
	"github.com/xzzpig/rclone-sync/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCtx() context.Context {
	return context.Background()
}

func setupImportTest(t *testing.T) (*gin.Engine, *services.ConnectionService) {
	gin.SetMode(gin.TestMode)

	// Setup in-memory database
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	// Setup encryptor
	encryptor, err := crypto.NewEncryptor("test-key-32-bytes-for-testing!!")
	require.NoError(t, err)

	// Create services
	connService := services.NewConnectionService(client, encryptor)

	// Setup router
	r := gin.New()

	handler := NewImportHandler(connService)
	r.POST("/import/parse", handler.Parse)
	r.POST("/import/execute", handler.Execute)

	return r, connService
}

// T067 [P] [US7] API 测试：POST /import/parse
func TestImportHandler_Parse(t *testing.T) {
	t.Run("parse valid rclone.conf content", func(t *testing.T) {
		r, _ := setupImportTest(t)

		content := `[test_local]
type = local

[test_s3]
type = s3
access_key_id = AKIAIOSFODNN7EXAMPLE
secret_access_key = secret
region = us-east-1
`
		reqBody := map[string]string{
			"content": content,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/parse", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		connections := resp["connections"].([]interface{})
		assert.Len(t, connections, 2)

		// Check first connection
		conn1 := connections[0].(map[string]interface{})
		assert.Equal(t, "test_local", conn1["name"])
		assert.Equal(t, "local", conn1["type"])

		// Check second connection
		conn2 := connections[1].(map[string]interface{})
		assert.Equal(t, "test_s3", conn2["name"])
		assert.Equal(t, "s3", conn2["type"])
	})

	t.Run("parse empty content returns empty list", func(t *testing.T) {
		r, _ := setupImportTest(t)

		reqBody := map[string]string{
			"content": "",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/parse", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		connections := resp["connections"].([]interface{})
		assert.Empty(t, connections)
	})

	t.Run("parse invalid content returns error", func(t *testing.T) {
		r, _ := setupImportTest(t)

		content := `[test_local]
name = test_local
path = /tmp
`
		reqBody := map[string]string{
			"content": content,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/parse", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("detect conflicts with existing connections", func(t *testing.T) {
		r, connService := setupImportTest(t)

		// Create existing connection
		_, err := connService.CreateConnection(testCtx(), "test_local", "local", map[string]string{"type": "local"})
		require.NoError(t, err)

		content := `[test_local]
type = local

[test_s3]
type = s3
`
		reqBody := map[string]string{
			"content": content,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/parse", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		validation := resp["validation"].(map[string]interface{})
		conflicts := validation["conflicts"].([]interface{})
		assert.Contains(t, conflicts, "test_local")

		valid := validation["valid"].([]interface{})
		assert.Len(t, valid, 1)
	})
}

// T068 [P] [US7] API 测试：POST /import/execute
func TestImportHandler_Execute(t *testing.T) {
	t.Run("import valid connections", func(t *testing.T) {
		r, connService := setupImportTest(t)

		connections := []map[string]interface{}{
			{
				"name": "test_local",
				"type": "local",
				"config": map[string]interface{}{
					"type": "local",
				},
			},
			{
				"name": "test_s3",
				"type": "s3",
				"config": map[string]interface{}{
					"type":              "s3",
					"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
					"secret_access_key": "secret",
					"region":            "us-east-1",
				},
			},
		}

		reqBody := map[string]interface{}{
			"connections": connections,
			"overwrite":   false,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, float64(2), resp["imported"])
		assert.Equal(t, float64(0), resp["skipped"])
		assert.Equal(t, float64(0), resp["failed"])

		// Verify connections were created
		conn, err := connService.GetConnectionByName(testCtx(), "test_local")
		require.NoError(t, err)
		assert.Equal(t, "local", conn.Type)

		conn, err = connService.GetConnectionByName(testCtx(), "test_s3")
		require.NoError(t, err)
		assert.Equal(t, "s3", conn.Type)
	})

	t.Run("skip conflicting connections without overwrite", func(t *testing.T) {
		r, connService := setupImportTest(t)

		// Create existing connection
		_, err := connService.CreateConnection(testCtx(), "test_local", "local", map[string]string{"type": "local"})
		require.NoError(t, err)

		connections := []map[string]interface{}{
			{
				"name": "test_local",
				"type": "local",
				"config": map[string]interface{}{
					"type": "local",
				},
			},
			{
				"name": "test_s3",
				"type": "s3",
				"config": map[string]interface{}{
					"type": "s3",
				},
			},
		}

		reqBody := map[string]interface{}{
			"connections": connections,
			"overwrite":   false,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, float64(1), resp["imported"]) // Only test_s3
		assert.Equal(t, float64(1), resp["skipped"])  // test_local skipped
		assert.Equal(t, float64(0), resp["failed"])
	})

	t.Run("overwrite existing connections with overwrite flag", func(t *testing.T) {
		r, connService := setupImportTest(t)

		// Create existing connection
		_, err := connService.CreateConnection(testCtx(), "test_local", "local", map[string]string{
			"type": "local",
			"path": "/old/path",
		})
		require.NoError(t, err)

		connections := []map[string]interface{}{
			{
				"name": "test_local",
				"type": "local",
				"config": map[string]interface{}{
					"type": "local",
					"path": "/new/path",
				},
			},
		}

		reqBody := map[string]interface{}{
			"connections": connections,
			"overwrite":   true,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, float64(1), resp["imported"])
		assert.Equal(t, float64(0), resp["skipped"])

		// Verify connection was updated
		_, err = connService.GetConnectionByName(testCtx(), "test_local")
		require.NoError(t, err)
		config, err := connService.GetConnectionConfig(testCtx(), "test_local")
		require.NoError(t, err)
		assert.Equal(t, "/new/path", config["path"])
	})

	t.Run("empty connections list returns error", func(t *testing.T) {
		r, _ := setupImportTest(t)

		reqBody := map[string]interface{}{
			"connections": []map[string]interface{}{},
			"overwrite":   false,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/import/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
