// Package context provides request context utilities for the API.
package context

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/config"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// authLog returns a named logger for the api.auth package.
func authLog() *zap.Logger {
	return logger.Named("api.auth")
}

// BasicAuthMiddleware creates a gin middleware that validates HTTP Basic Auth credentials.
// It uses constant-time comparison for password comparison to prevent timing attacks.
// On successful authentication, the username is stored in the gin context using gin.AuthUserKey.
func BasicAuthMiddleware(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract credentials from Authorization header
		user, pass, hasAuth := c.Request.BasicAuth()

		// Validate credentials using constant-time comparison
		// This prevents timing attacks that could reveal password information
		userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1
		authValid := hasAuth && userMatch && passMatch

		if !authValid {
			// Set WWW-Authenticate header to prompt browser for credentials
			c.Header("WWW-Authenticate", `Basic realm="Login Required"`)

			// Log authentication failure (never log passwords)
			authLog().Warn("authentication failed",
				zap.String("ip", c.ClientIP()),
				zap.String("username", user),
				zap.String("path", c.Request.URL.Path),
			)

			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Store authenticated username in context for downstream handlers
		c.Set(gin.AuthUserKey, username)

		c.Next()
	}
}

// OptionalAuthMiddleware returns a conditional authentication middleware.
// If authentication is enabled in the config, it returns the BasicAuthMiddleware.
// If authentication is disabled, it returns a no-op middleware that passes all requests.
func OptionalAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	if cfg.IsAuthEnabled() {
		return BasicAuthMiddleware(cfg.Auth.Username, cfg.Auth.Password)
	}
	// Return no-op middleware when auth is disabled
	return func(c *gin.Context) {
		c.Next()
	}
}
