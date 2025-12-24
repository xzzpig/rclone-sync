// Package subscription provides GraphQL subscription infrastructure.
package subscription

import (
	"sync"

	"github.com/google/uuid"
)

// GenericSubscriber represents a generic subscription.
type GenericSubscriber[T any] struct {
	// ID is the unique identifier for this subscriber.
	ID string
	// Filter is an optional function that filters events.
	// If nil, all events are accepted.
	// Returns true to accept the event, false to reject.
	Filter func(T) bool
	// Events is the channel where events are delivered.
	Events chan T
}

// GenericEventBus is a generic event bus that supports type-safe subscriptions.
type GenericEventBus[T any] struct {
	mu          sync.RWMutex
	subscribers map[string]*GenericSubscriber[T]
	bufferSize  int
}

// NewGenericEventBus creates a new generic event bus.
// bufferSize specifies the buffer size for subscriber channels.
func NewGenericEventBus[T any](bufferSize int) *GenericEventBus[T] {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &GenericEventBus[T]{
		subscribers: make(map[string]*GenericSubscriber[T]),
		bufferSize:  bufferSize,
	}
}

// Subscribe creates a new subscription with an optional filter.
// If filter is nil, all events are delivered to this subscriber.
func (eb *GenericEventBus[T]) Subscribe(filter func(T) bool) *GenericSubscriber[T] {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	sub := &GenericSubscriber[T]{
		ID:     uuid.New().String(),
		Filter: filter,
		Events: make(chan T, eb.bufferSize),
	}
	eb.subscribers[sub.ID] = sub
	return sub
}

// Unsubscribe removes a subscription by ID.
// The subscriber's Events channel is closed.
func (eb *GenericEventBus[T]) Unsubscribe(subscriberID string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if sub, ok := eb.subscribers[subscriberID]; ok {
		close(sub.Events)
		delete(eb.subscribers, subscriberID)
	}
}

// Publish sends an event to all matching subscribers.
// Events are filtered by each subscriber's Filter function.
// Non-blocking: if a subscriber's buffer is full, the event is dropped for that subscriber.
func (eb *GenericEventBus[T]) Publish(event T) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for _, sub := range eb.subscribers {
		// Apply filter if present
		if sub.Filter != nil && !sub.Filter(event) {
			continue
		}

		// Non-blocking send
		select {
		case sub.Events <- event:
			// Event sent successfully
		default:
			// Channel buffer full, drop event for this subscriber
		}
	}
}

// SubscriberCount returns the number of active subscribers.
func (eb *GenericEventBus[T]) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.subscribers)
}
