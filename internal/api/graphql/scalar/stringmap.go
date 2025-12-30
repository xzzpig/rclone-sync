// Package scalar provides custom GraphQL scalar implementations.
package scalar

import (
	"encoding/json"
	"io"

	"github.com/99designs/gqlgen/graphql"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

// ConstError is an alias to errs.ConstError for defining sentinel errors in this package.
type ConstError = errs.ConstError

const (
	// ErrStringMapInvalidType is returned when the input is not a JSON object.
	ErrStringMapInvalidType ConstError = "StringMap must be a JSON object"
	// ErrStringMapValueNotString is returned when a value in the map is not a string.
	ErrStringMapValueNotString ConstError = "StringMap value must be string"
)

// StringMap is a type alias for map[string]string, used for gqlgen model binding.
// It represents a JSON object where all values must be strings.
type StringMap map[string]string

// MarshalStringMap marshals a map[string]string to a GraphQL JSON object.
func MarshalStringMap(val map[string]string) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_ = json.NewEncoder(w).Encode(val)
	})
}

// UnmarshalStringMap unmarshals a GraphQL JSON object to a map[string]string.
// It returns an error if the input is not a JSON object or if any value is not a string.
func UnmarshalStringMap(v interface{}) (map[string]string, error) {
	if v == nil {
		return nil, nil
	}

	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, ErrStringMapInvalidType
	}

	result := make(map[string]string, len(m))
	for k, val := range m {
		str, ok := val.(string)
		if !ok {
			return nil, ErrStringMapValueNotString
		}
		result[k] = str
	}
	return result, nil
}
