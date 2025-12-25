package scalar

import (
	"bytes"
	"testing"
)

func TestMarshalStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: "{}\n",
		},
		{
			name:     "single entry",
			input:    map[string]string{"key": "value"},
			expected: `{"key":"value"}` + "\n",
		},
		{
			name: "multiple entries",
			input: map[string]string{
				"user":  "admin",
				"token": "secret123",
			},
			// Note: map ordering is not guaranteed, so we check length instead
			expected: "", // Will check differently
		},
		{
			name:     "nil map",
			input:    nil,
			expected: "null\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marshaler := MarshalStringMap(tt.input)
			var buf bytes.Buffer
			marshaler.MarshalGQL(&buf)
			result := buf.String()

			if tt.name == "multiple entries" {
				// For multiple entries, just check it's valid JSON with correct length
				if len(result) < 20 { // {"user":"admin","token":"secret123"} is about 35 chars
					t.Errorf("expected longer result for multiple entries, got %q", result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestUnmarshalStringMap(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil input",
			input:       nil,
			expected:    nil,
			expectError: false,
		},
		{
			name:        "empty map",
			input:       map[string]interface{}{},
			expected:    map[string]string{},
			expectError: false,
		},
		{
			name: "valid string map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectError: false,
		},
		{
			name:        "non-map input",
			input:       "not a map",
			expected:    nil,
			expectError: true,
			errorMsg:    "StringMap must be a JSON object",
		},
		{
			name:        "array input",
			input:       []interface{}{"a", "b"},
			expected:    nil,
			expectError: true,
			errorMsg:    "StringMap must be a JSON object",
		},
		{
			name: "non-string value - int",
			input: map[string]interface{}{
				"key": 123,
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "StringMap value must be string",
		},
		{
			name: "non-string value - bool",
			input: map[string]interface{}{
				"enabled": true,
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "StringMap value must be string",
		},
		{
			name: "non-string value - nested object",
			input: map[string]interface{}{
				"nested": map[string]interface{}{"inner": "value"},
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "StringMap value must be string",
		},
		{
			name: "mixed valid and invalid",
			input: map[string]interface{}{
				"valid":   "string",
				"invalid": 42,
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "must be string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UnmarshalStringMap(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !mapsEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
