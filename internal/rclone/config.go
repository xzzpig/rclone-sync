package rclone

import (
	"context"
	"fmt"
	"slices"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	appConfig "github.com/xzzpig/rclone-sync/internal/core/config"
)

// InitConfig initializes the rclone configuration.
func InitConfig(configPath string) {
	config.SetConfigPath(configPath)
	configfile.Install()

	// Set rclone log level based on app config
	switch appConfig.Cfg.Log.Level {
	case "debug":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelDebug
	case "info":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelInfo
	case "warn":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelNotice
	case "error":
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelError
	default:
		fs.GetConfig(context.Background()).LogLevel = fs.LogLevelNotice
	}
}

// ListRemotes lists all configured rclone remotes.
func ListRemotes() []string {
	return config.GetRemoteNames()
}

// RemoteInfo holds the configuration for a remote.
type RemoteInfo struct {
	Type   string `json:"type,omitempty"`
	Remote string `json:"remote,omitempty"`
}

// GetRemoteInfo gets all parameters for a given remote.
func GetRemoteInfo(remoteName string) (*RemoteInfo, error) {
	sections := config.FileSections()

	remoteExists := slices.Contains(sections, remoteName)

	if !remoteExists {
		return nil, fmt.Errorf("remote %q not found", remoteName)
	}

	info := &RemoteInfo{}

	// This is a workaround as there is no direct `GetSection`
	// We would need to parse the config file manually for a more robust solution
	// For now, we can get known keys like 'type'
	if val, ok := config.FileGetValue(remoteName, "type"); ok {
		info.Type = val
	}
	if val, ok := config.FileGetValue(remoteName, "remote"); ok {
		info.Remote = val
	}
	// Add other common keys if needed

	return info, nil
}

// CreateRemote creates or updates a remote with the given parameters.
func CreateRemote(remoteName string, params map[string]string) error {
	for key, value := range params {
		config.FileSetValue(remoteName, key, value)
	}
	config.SaveConfig()
	return nil
}

// DeleteRemote deletes a remote.
func DeleteRemote(remoteName string) {
	config.DeleteRemote(remoteName)
}
