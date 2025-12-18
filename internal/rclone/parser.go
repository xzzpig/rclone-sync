package rclone

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/unknwon/goconfig"
)

// ParsedConnection represents a connection parsed from rclone.conf.
type ParsedConnection struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// ValidationResult contains the validation results for import
type ValidationResult struct {
	Valid              []ParsedConnection `json:"valid"`
	Conflicts          []string           `json:"conflicts"`
	InternalDuplicates []string           `json:"internal_duplicates"`
}

// ParseRcloneConf parses rclone.conf content using rclone's own dependency (goconfig).
func ParseRcloneConf(content string) ([]ParsedConnection, error) {
	// Handle empty content
	content = strings.TrimSpace(content)
	if content == "" {
		return []ParsedConnection{}, nil
	}

	// Parse using goconfig (rclone's own dependency)
	cfg, err := goconfig.LoadFromReader(bytes.NewReader([]byte(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rclone.conf format: %w", err)
	}

	var connections []ParsedConnection

	// Iterate through sections (each section is a connection)
	for _, section := range cfg.GetSectionList() {
		// Skip the default section
		if section == "" || section == "DEFAULT" {
			continue
		}

		config := make(map[string]string)

		// Extract all key-value pairs
		for _, key := range cfg.GetKeyList(section) {
			if value, err := cfg.GetValue(section, key); err == nil {
				config[key] = value
			}
		}

		// Validate required fields
		connType, ok := config["type"]
		if !ok || connType == "" {
			return nil, fmt.Errorf("connection '%s' missing required field 'type'", section) //nolint:err113
		}

		connections = append(connections, ParsedConnection{
			Name:   section,
			Type:   connType,
			Config: config,
		})
	}

	return connections, nil
}

// ValidateImport validates parsed connections against existing connections
// and detects internal duplicates
func ValidateImport(parsed []ParsedConnection, existing []string) *ValidationResult {
	result := &ValidationResult{
		Valid:              []ParsedConnection{},
		Conflicts:          []string{},
		InternalDuplicates: []string{},
	}

	// Build a set of existing connection names
	existingSet := make(map[string]bool)
	for _, name := range existing {
		existingSet[name] = true
	}

	// Track names seen in parsed connections to detect internal duplicates
	seenNames := make(map[string]bool)
	duplicateNames := make(map[string]bool)

	// First pass: detect internal duplicates
	for _, conn := range parsed {
		if seenNames[conn.Name] {
			duplicateNames[conn.Name] = true
		}
		seenNames[conn.Name] = true
	}

	// Second pass: categorize connections
	for _, conn := range parsed {
		// Skip if it's an internal duplicate
		if duplicateNames[conn.Name] {
			if !contains(result.InternalDuplicates, conn.Name) {
				result.InternalDuplicates = append(result.InternalDuplicates, conn.Name)
			}
			continue
		}

		// Check if it conflicts with existing connections
		if existingSet[conn.Name] {
			result.Conflicts = append(result.Conflicts, conn.Name)
			continue
		}

		// Valid connection
		result.Valid = append(result.Valid, conn)
	}

	return result
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
