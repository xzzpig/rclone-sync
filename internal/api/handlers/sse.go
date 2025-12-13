package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/api/sse"
)

type SSEHandler struct {
}

// GetGlobalEvents streams all events, optionally filtered by parameters
func GetGlobalEvents(c *gin.Context) {
	eventName := c.Query("event")
	remoteName := c.Query("remote_name")

	broker := sse.GetBroker()
	clientChan := broker.Subscribe()
	defer broker.Unsubscribe(clientChan)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-clientChan:
			if !ok {
				return false
			}

			// If event filter is specified, check event type
			if eventName != "" && event.Type != eventName {
				return true // Skip this event
			}

			// If remote_name filter is specified, check if event belongs to that remote
			if remoteName != "" {
				// Check if event data contains remote_name field
				if data, ok := event.Data.(gin.H); ok {
					if rName, exists := data["remote_name"]; exists {
						if rName != remoteName {
							return true // Skip this event
						}
					}
				} else if data, ok := event.Data.(map[string]any); ok {
					if rName, exists := data["remote_name"]; exists {
						if rName != remoteName {
							return true // Skip this event
						}
					}
				}
			}

			c.SSEvent(event.Type, event.Data)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// GetConnectionEvents streams events filtered by connection name
func GetConnectionEvents(c *gin.Context) {
	connectionName := c.Param("name")
	if connectionName == "" {
		HandleError(c, NewError(http.StatusBadRequest, "Connection name required", ""))
		return
	}

	broker := sse.GetBroker()
	clientChan := broker.Subscribe()
	defer broker.Unsubscribe(clientChan)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-clientChan:
			if !ok {
				return false
			}

			// Filter events based on connection name by checking for remote_name in event data
			if data, ok := event.Data.(gin.H); ok {
				if rName, exists := data["remote_name"]; exists {
					if rName != connectionName {
						return true // Skip this event
					}
				}
			} else if data, ok := event.Data.(map[string]any); ok {
				if rName, exists := data["remote_name"]; exists {
					if rName != connectionName {
						return true // Skip this event
					}
				}
			}

			c.SSEvent(event.Type, event.Data)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
