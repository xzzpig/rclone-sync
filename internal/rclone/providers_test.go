package rclone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

func TestListProviders(t *testing.T) {
	providers := rclone.ListProviders()

	// Should have many providers
	assert.NotEmpty(t, providers)

	// Verify memory provider exists
	found := false
	for _, p := range providers {
		if p.Name == "memory" {
			found = true
			assert.NotEmpty(t, p.Description)
			break
		}
	}
	assert.True(t, found, "memory provider should be in the list")
}

func TestGetProviderOptions_Memory(t *testing.T) {
	provider, err := rclone.GetProviderOptions("memory")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "memory", provider.Name)
	assert.NotEmpty(t, provider.Description)
	// Memory provider has options
	assert.NotNil(t, provider.Options)
}

func TestGetProviderOptions_Local(t *testing.T) {
	provider, err := rclone.GetProviderOptions("local")
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "local", provider.Name)
	assert.NotEmpty(t, provider.Description)
	assert.NotNil(t, provider.Options)
}

func TestGetProviderOptions_Unknown(t *testing.T) {
	_, err := rclone.GetProviderOptions("non-existent-provider")
	assert.Error(t, err)
}

func TestGetProviderOptions_CommonProviders(t *testing.T) {
	// Test some common providers
	commonProviders := []string{
		"local",
		"memory",
		"s3",
		"drive",
		"dropbox",
	}

	for _, providerName := range commonProviders {
		t.Run(providerName, func(t *testing.T) {
			provider, err := rclone.GetProviderOptions(providerName)
			require.NoError(t, err, "Provider %s should exist", providerName)
			assert.Equal(t, providerName, provider.Name)
			assert.NotEmpty(t, provider.Description)
		})
	}
}
