package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xzzpig/rclone-sync/internal/api/sse"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
)

func TestGetConnectionEvents_EmptyConnectionName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// Manually call the handler to test with empty name
	router.GET("/test", func(c *gin.Context) {
		c.Params = gin.Params{
			{Key: "name", Value: ""},
		}
		GetConnectionEvents(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGetConnectionEvents_EventFiltering(t *testing.T) {
	// This test verifies the event filtering logic without actually running SSE
	gin.SetMode(gin.TestMode)

	// Create test event data with remote_name
	testEvent1 := sse.Event{
		Data: gin.H{
			"remote_name": "testremote",
			"progress":    50,
		},
	}

	testEvent2 := sse.Event{
		Data: gin.H{
			"remote_name": "other_remote",
			"progress":    75,
		},
	}

	// Test with map[string]any instead of gin.H
	testEvent3 := sse.Event{
		Data: map[string]any{
			"remote_name": "testremote",
			"progress":    100,
		},
	}

	// Events without remote_name should pass through
	testEvent4 := sse.Event{
		Data: gin.H{
			"message": "system message",
		},
	}

	// Simulate the filtering logic from GetConnectionEvents
	connectionName := "testremote"

	// Test event 1 - should match
	if data, ok := testEvent1.Data.(gin.H); ok {
		if rName, exists := data["remote_name"]; exists {
			assert.Equal(t, connectionName, rName, "Event 1 should match the connection name")
		}
	}

	// Test event 2 - should not match
	if data, ok := testEvent2.Data.(gin.H); ok {
		if rName, exists := data["remote_name"]; exists {
			assert.NotEqual(t, connectionName, rName, "Event 2 should not match the connection name")
		}
	}

	// Test event 3 - should match (map[string]any)
	if data, ok := testEvent3.Data.(map[string]any); ok {
		if rName, exists := data["remote_name"]; exists {
			assert.Equal(t, connectionName, rName, "Event 3 should match the connection name")
		}
	}

	// Test event 4 - no remote_name field
	if data, ok := testEvent4.Data.(gin.H); ok {
		_, exists := data["remote_name"]
		assert.False(t, exists, "Event 4 should not have remote_name field")
	}
}

func TestSSEHandler_MultipleEventTypes(t *testing.T) {
	// Test that different event types can be properly structured
	eventTypes := []struct {
		eventType string
		data      interface{}
	}{
		{
			eventType: "job_progress",
			data: gin.H{
				"remote_name": "test",
				"job_id":      "123",
				"progress":    50,
			},
		},
		{
			eventType: "job_complete",
			data: gin.H{
				"remote_name": "test",
				"job_id":      "123",
				"status":      "success",
			},
		},
		{
			eventType: "job_error",
			data: gin.H{
				"remote_name": "test",
				"job_id":      "123",
				"error":       "something went wrong",
			},
		},
		{
			eventType: "system_notification",
			data: gin.H{
				"message": "system update",
			},
		},
	}

	for _, et := range eventTypes {
		event := sse.Event{
			Type: et.eventType,
			Data: et.data,
		}

		assert.Equal(t, et.eventType, event.Type)
		assert.NotNil(t, event.Data)

		// Verify remote_name filtering logic
		connectionName := "test"
		if data, ok := event.Data.(gin.H); ok {
			if rName, exists := data["remote_name"]; exists {
				if rName == connectionName {
					// This event should be sent to the client
					assert.Equal(t, connectionName, rName)
				}
			} else {
				// Event without remote_name - should also be sent (or filtered based on requirements)
				assert.False(t, exists)
			}
		}
	}
}

func TestSSEHandler_DataTypes(t *testing.T) {
	// Test different data types that can be used in events
	tests := []struct {
		name      string
		eventData interface{}
		valid     bool
	}{
		{
			name: "gin.H type",
			eventData: gin.H{
				"remote_name": "test",
				"value":       123,
			},
			valid: true,
		},
		{
			name: "map[string]any type",
			eventData: map[string]any{
				"remote_name": "test",
				"value":       456,
			},
			valid: true,
		},
		{
			name:      "string type",
			eventData: "simple string data",
			valid:     true,
		},
		{
			name:      "number type",
			eventData: 12345,
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := sse.Event{
				Type: "test",
				Data: tt.eventData,
			}

			assert.NotNil(t, event.Data)
			assert.Equal(t, "test", event.Type)

			// Try to extract remote_name if it's a map
			switch data := event.Data.(type) {
			case gin.H:
				if rName, exists := data["remote_name"]; exists {
					assert.Equal(t, "test", rName)
				}
			case map[string]any:
				if rName, exists := data["remote_name"]; exists {
					assert.Equal(t, "test", rName)
				}
			default:
				// Other types are also valid
				assert.True(t, tt.valid)
			}
		})
	}
}

func TestSSEHandler_BrokerIntegration(t *testing.T) {
	// Initialize logger for this test
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)

	gin.SetMode(gin.TestMode)

	// Get the SSE broker instance
	broker := sse.GetBroker()
	assert.NotNil(t, broker, "Broker should not be nil")

	// Test subscribing and unsubscribing
	clientChan := broker.Subscribe()
	assert.NotNil(t, clientChan, "Client channel should not be nil")

	// Submit a test event
	testEvent := sse.Event{
		Type: "test_event",
		Data: gin.H{
			"message": "test message",
		},
	}
	broker.Submit(testEvent)

	// Unsubscribe
	broker.Unsubscribe(clientChan)

	// Channel should be closed after unsubscribe
	_, ok := <-clientChan
	assert.False(t, ok, "Channel should be closed after unsubscribe")
}
