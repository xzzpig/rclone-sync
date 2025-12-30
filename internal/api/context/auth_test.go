package context

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xzzpig/rclone-sync/internal/core/config"
)

// basicAuthEncode encodes username and password for Basic Auth header
func basicAuthEncode(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestBasicAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		username            string
		password            string
		authHeader          string
		expectedStatusCode  int
		expectedWWWAuth     string
		expectedBodyContent string
	}{
		{
			name:               "No auth header returns 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "",
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
		{
			name:               "Invalid credentials return 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "Basic " + basicAuthEncode("wronguser", "wrongpass"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
		{
			name:                "Valid credentials pass through",
			username:            "admin",
			password:            "secret123",
			authHeader:          "Basic " + basicAuthEncode("admin", "secret123"),
			expectedStatusCode:  http.StatusOK,
			expectedWWWAuth:     "",
			expectedBodyContent: `"message":"success"`,
		},
		{
			name:               "Wrong password returns 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "Basic " + basicAuthEncode("admin", "wrongpassword"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
		{
			name:               "Wrong username returns 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "Basic " + basicAuthEncode("wronguser", "secret123"),
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
		{
			name:               "Only username without password returns 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "Basic YWRtaW4=", // base64 of "admin" without colon
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
		{
			name:               "Only password without username returns 401",
			username:           "admin",
			password:           "secret123",
			authHeader:         "Basic OnNlY3JldDEyMw==", // base64 of ":secret123"
			expectedStatusCode: http.StatusUnauthorized,
			expectedWWWAuth:    `Basic realm="Login Required"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test router with auth middleware
			router := gin.New()
			router.Use(BasicAuthMiddleware(tt.username, tt.password))
			router.GET("/test", func(c *gin.Context) {
				username := c.GetString(gin.AuthUserKey)
				c.JSON(200, gin.H{"message": "success", "user": username})
			})

			// Make a request
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedWWWAuth != "" {
				assert.Equal(t, tt.expectedWWWAuth, w.Header().Get("WWW-Authenticate"))
			}
			if tt.expectedBodyContent != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContent)
			}
		})
	}
}

func TestOptionalAuthMiddleware_AuthDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create config with auth disabled
	cfg := &config.Config{}
	cfg.Auth.Username = ""
	cfg.Auth.Password = ""

	// Create a test router with optional auth middleware
	router := gin.New()
	router.Use(OptionalAuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make a request without auth header (should pass through)
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify 200 response (auth is disabled, so request passes through)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("WWW-Authenticate"))
}

func TestOptionalAuthMiddleware_AuthEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create config with auth enabled
	cfg := &config.Config{}
	cfg.Auth.Username = "admin"
	cfg.Auth.Password = "secret123"

	// Create a test router with optional auth middleware
	router := gin.New()
	router.Use(OptionalAuthMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		username := c.GetString(gin.AuthUserKey)
		c.JSON(200, gin.H{"message": "success", "user": username})
	})

	// Test without credentials - should be rejected
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusUnauthorized, w1.Code)

	// Test with valid credentials - should pass
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.SetBasicAuth("admin", "secret123")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}
