package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestRemoteAPI_GetProviderOptions_Success(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Test with local provider which always exists
	resp, err := http.Get(ts.Server.URL + "/api/providers/local")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var options interface{}
	err = json.NewDecoder(resp.Body).Decode(&options)
	require.NoError(t, err)
	assert.NotNil(t, options)
}

func TestRemoteAPI_GetProviderOptions_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Test with non-existent provider
	resp, err := http.Get(ts.Server.URL + "/api/providers/nonexistent-provider-xyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
