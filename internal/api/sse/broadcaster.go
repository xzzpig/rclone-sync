// Package sse provides Server-Sent Events functionality for real-time updates.
package sse

import (
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
)

// Event represents a Server-Sent Event with a type and data payload.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Broadcaster manages SSE client subscriptions and event distribution.
type Broadcaster struct {
	clients   map[chan Event]bool
	mu        sync.RWMutex
	logger    *zap.Logger
	stopChan  chan struct{}
	eventChan chan Event
}

var broker *Broadcaster
var once sync.Once

// GetBroker returns the singleton instance of the Broadcaster.
// It initializes the broker on the first call.
func GetBroker() *Broadcaster {
	once.Do(func() {
		broker = NewBroadcaster()
		go broker.run()
	})
	return broker
}

// NewBroadcaster creates a new Broadcaster instance.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients:   make(map[chan Event]bool),
		logger:    logger.L.Named("sse-broker"),
		stopChan:  make(chan struct{}),
		eventChan: make(chan Event, 100), // Buffered channel
	}
}

func (b *Broadcaster) run() {
	for {
		select {
		case event := <-b.eventChan:
			b.broadcast(event)
		case <-b.stopChan:
			return
		}
	}
}

// Subscribe registers a new client and returns a channel for receiving events.
func (b *Broadcaster) Subscribe() chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	client := make(chan Event, 10) // Buffered channel for each client
	b.clients[client] = true
	b.logger.Info("New client subscribed")
	return client
}

// Unsubscribe removes a client from the broadcaster.
func (b *Broadcaster) Unsubscribe(client chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.clients[client]; ok {
		delete(b.clients, client)
		close(client)
		b.logger.Info("Client unsubscribed")
	}
}

// Submit sends an event to all subscribed clients.
func (b *Broadcaster) Submit(event Event) {
	select {
	case b.eventChan <- event:
	default:
		b.logger.Warn("Event channel is full, dropping event", zap.Any("event", event))
	}
}

func (b *Broadcaster) broadcast(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for client := range b.clients {
		select {
		case client <- event:
		default:
			// Client channel is full, maybe log this
		}
	}
}

// Stop terminates the broadcaster's event loop.
func (b *Broadcaster) Stop() {
	close(b.stopChan)
}

// Handler is a Gin handler function for SSE endpoints.
func Handler(c *gin.Context) {
	clientChan := GetBroker().Subscribe()
	defer GetBroker().Unsubscribe(clientChan)

	c.Stream(func(_ io.Writer) bool {
		select {
		case event, ok := <-clientChan:
			if !ok {
				return false // Channel closed
			}
			c.SSEvent(event.Type, event.Data)
			return true
		case <-c.Request.Context().Done():
			return false // Client disconnected
		}
	})
}
