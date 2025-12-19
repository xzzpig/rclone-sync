// Package rclone provides rclone integration functionality.
// This file implements config.Storage interface to store rclone configuration in database.
package rclone

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/fs/config"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
)

// DBStorage implements config.Storage interface for database-backed configuration storage.
// This allows rclone to read/write configuration directly from/to the database,
// enabling automatic token refresh persistence.
type DBStorage struct {
	svc ports.ConnectionService
	mu  sync.RWMutex
}

// NewDBStorage creates a new database-backed storage instance.
func NewDBStorage(svc ports.ConnectionService) *DBStorage {
	return &DBStorage{
		svc: svc,
	}
}

// Install sets this DBStorage as the active rclone configuration storage.
// This should be called during application startup, after ConnectionService is initialized.
// Note: Do NOT call configfile.Install() when using DBStorage.
func (s *DBStorage) Install() {
	config.SetData(s)
}

// GetSectionList returns a list of all connection names (sections).
func (s *DBStorage) GetSectionList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	names, err := s.svc.ListConnectionNames(ctx)
	if err != nil {
		return nil
	}
	return names
}

// HasSection checks if a connection with the given name exists.
func (s *DBStorage) HasSection(section string) bool {
	if section == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	_, err := s.svc.GetConnectionByName(ctx, section)
	return err == nil
}

// DeleteSection removes a connection and clears its cache.
func (s *DBStorage) DeleteSection(section string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	_ = s.svc.DeleteConnectionByName(ctx, section)

	// Clear rclone cache for this remote
	cache.ClearConfig(section)
}

// GetKeyList returns all configuration keys for a connection.
func (s *DBStorage) GetKeyList(section string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	cfg, err := s.svc.GetConnectionConfig(ctx, section)
	if err != nil {
		return nil
	}

	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	return keys
}

// GetValue retrieves a configuration value for a connection.
func (s *DBStorage) GetValue(section, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	cfg, err := s.svc.GetConnectionConfig(ctx, section)
	if err != nil {
		return "", false
	}

	value, ok := cfg[key]
	return value, ok
}

// SetValue sets a configuration value for a connection.
// If the connection doesn't exist, it creates a new one.
// This is called by rclone when refreshing OAuth tokens.
func (s *DBStorage) SetValue(section, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Try to get existing connection
	conn, err := s.svc.GetConnectionByName(ctx, section)
	if err != nil {
		// Connection doesn't exist, create a new one
		cfg := map[string]string{key: value}
		connType := ""
		if key == "type" {
			connType = value
		}
		_, _ = s.svc.CreateConnection(ctx, section, connType, cfg)
		return
	}

	// Get current config
	cfg, err := s.svc.GetConnectionConfig(ctx, section)
	if err != nil {
		cfg = make(map[string]string)
	}

	// Update the value
	cfg[key] = value

	// Determine connection type
	connType := conn.Type
	if key == "type" {
		connType = value
	}

	// Update the connection
	_ = s.svc.UpdateConnection(ctx, conn.ID, nil, &connType, cfg)

	// Clear cache so rclone reloads the config
	cache.ClearConfig(section)
}

// DeleteKey removes a configuration key from a connection.
func (s *DBStorage) DeleteKey(section, key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Get existing connection
	conn, err := s.svc.GetConnectionByName(ctx, section)
	if err != nil {
		return false
	}

	// Get current config
	cfg, err := s.svc.GetConnectionConfig(ctx, section)
	if err != nil {
		return false
	}

	// Check if key exists
	if _, ok := cfg[key]; !ok {
		return false
	}

	// Remove the key
	delete(cfg, key)

	// Update the connection
	_ = s.svc.UpdateConnection(ctx, conn.ID, nil, nil, cfg)

	// Clear cache
	cache.ClearConfig(section)

	return true
}

// Load is a no-op for database storage.
// Data is loaded on-demand from the database.
func (s *DBStorage) Load() error {
	return nil
}

// Save is a no-op for database storage.
// Data is persisted immediately on each SetValue call.
func (s *DBStorage) Save() error {
	return nil
}

// Serialize returns all connections as JSON.
// This is used by rclone config show --json and similar commands.
func (s *DBStorage) Serialize() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	conns, err := s.svc.ListConnections(ctx)
	if err != nil {
		return "{}", nil
	}

	if len(conns) == 0 {
		return "{}", nil
	}

	result := make(map[string]map[string]string)
	for _, c := range conns {
		cfg, err := s.svc.GetConnectionConfig(ctx, c.Name)
		if err != nil {
			continue
		}
		result[c.Name] = cfg
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "{}", err
	}

	return string(data), nil
}

// Ensure DBStorage implements config.Storage interface
var _ config.Storage = (*DBStorage)(nil)
