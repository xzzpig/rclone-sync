package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteAPI_CreateAndList(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// 1. Create a remote via POST /remotes/:name
	remoteName := "testlocal"
	remotePayload := map[string]string{
		"type": "local",
	}
	payloadBytes, _ := json.Marshal(remotePayload)

	resp, err := http.Post(ts.Server.URL+"/api/remotes/"+remoteName, "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 2. List remotes via GET /remotes
	resp, err = http.Get(ts.Server.URL + "/api/remotes")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var remotes []map[string]any
	err = json.NewDecoder(resp.Body).Decode(&remotes)
	require.NoError(t, err)

	var remoteNames []string
	for _, remote := range remotes {
		remoteNames = append(remoteNames, remote["name"].(string))
	}
	assert.Contains(t, remoteNames, remoteName)

	// 3. Verify the rclone.conf file was updated
	rcloneConfPath := filepath.Join(ts.AppDataDir, "rclone.conf")
	content, err := os.ReadFile(rcloneConfPath)
	require.NoError(t, err)

	// Verify the configuration file contains the newly created remote
	assert.Contains(t, string(content), fmt.Sprintf("[%s]", remoteName))
	assert.Contains(t, string(content), "type = local")
}

func TestRemoteAPI_GetRemoteInfo(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create a remote first
	remoteName := "inforemote"
	remotePayload := map[string]string{"type": "local"}
	payloadBytes, _ := json.Marshal(remotePayload)
	resp, err := http.Post(ts.Server.URL+"/api/remotes/"+remoteName, "application/json", bytes.NewBuffer(payloadBytes))
	require.NoError(t, err)
	resp.Body.Close()

	// Get info
	resp, err = http.Get(ts.Server.URL + "/api/remotes/" + remoteName)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var info map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&info)
	require.NoError(t, err)
	assert.Equal(t, "local", info["type"])
}

func TestRemoteAPI_DeleteRemote(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create
	remoteName := "deleteremote"
	remotePayload := map[string]string{"type": "local"}
	payloadBytes, _ := json.Marshal(remotePayload)
	http.Post(ts.Server.URL+"/api/remotes/"+remoteName, "application/json", bytes.NewBuffer(payloadBytes))

	// Delete
	req, err := http.NewRequest(http.MethodDelete, ts.Server.URL+"/api/remotes/"+remoteName, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify gone
	resp, err = http.Get(ts.Server.URL + "/api/remotes/" + remoteName)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRemoteAPI_ListProviders(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	resp, err := http.Get(ts.Server.URL + "/api/providers")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var providers []interface{}
	err = json.NewDecoder(resp.Body).Decode(&providers)
	require.NoError(t, err)
	assert.NotEmpty(t, providers)
}

func TestRemoteAPI_Errors(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Get non-existent remote
	resp, err := http.Get(ts.Server.URL + "/api/remotes/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
