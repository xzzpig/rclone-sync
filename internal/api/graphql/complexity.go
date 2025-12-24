// Package graphql provides the GraphQL handler and complexity limits.
package graphql

import (
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
)

const (
	// DefaultComplexityLimit is the default maximum complexity allowed for a query.
	// Each field has a default cost of 1, so this limits the total number of fields.
	// This can be adjusted based on performance requirements.
	DefaultComplexityLimit = 200
)

// ConfigureComplexityLimit adds the fixed complexity limit extension to the GraphQL server.
// This helps prevent overly complex queries that could impact performance.
func ConfigureComplexityLimit(srv *handler.Server, limit int) {
	srv.Use(extension.FixedComplexityLimit(limit))
}
