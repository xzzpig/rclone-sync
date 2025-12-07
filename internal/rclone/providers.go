package rclone

import (
	_ "github.com/rclone/rclone/backend/all" // Import all backends
	"github.com/rclone/rclone/fs"
)

type Provider struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Options     []fs.Option `json:"options"`
}

// ListProviders lists all available rclone providers.
func ListProviders() []Provider {
	var providers []Provider
	for _, item := range fs.Registry {
		providers = append(providers, Provider{
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return providers
}

// GetProviderOptions gets all options for a given provider.
func GetProviderOptions(providerName string) (*Provider, error) {
	reg, err := fs.Find(providerName)
	if err != nil {
		return nil, err
	}
	return &Provider{
		Name:        reg.Name,
		Description: reg.Description,
		Options:     reg.Options,
	}, nil
}
