// Package graphql provides the GraphQL handler and playground.
package graphql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/generated"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
)

// NewHandler creates a new GraphQL handler with all transports configured.
func NewHandler(deps *resolver.Dependencies) *handler.Server {
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver.New(deps),
	}))

	// Configure error handling for i18n support
	srv.SetErrorPresenter(ErrorPresenter)
	srv.SetRecoverFunc(RecoverFunc)

	// Configure transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// WebSocket configuration for subscriptions
	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: restrict origin in production
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	// Enable query caching
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	// Enable automatic persisted queries
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Add logging extension
	srv.Use(NewLoggingExtension())

	// Enable introspection (for development)
	srv.Use(extension.Introspection{})

	// Configure query complexity limit to prevent expensive queries
	ConfigureComplexityLimit(srv, DefaultComplexityLimit)

	return srv
}

// GinHandler wraps the GraphQL handler for Gin compatibility.
func GinHandler(srv *handler.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		srv.ServeHTTP(c.Writer, c.Request)
	}
}

// PlaygroundHandler returns a handler for GraphiQL playground.
func PlaygroundHandler(endpoint string) gin.HandlerFunc {
	h := playground.Handler("GraphQL Playground", endpoint)
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
