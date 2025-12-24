package subscription

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvent is a simple event type for testing.
type TestEvent struct {
	ID      int
	Message string
	Tag     string
}

func TestNewGenericEventBus(t *testing.T) {
	t.Run("creates with specified buffer size", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](50)
		require.NotNil(t, bus)
		assert.Equal(t, 50, bus.bufferSize)
	})

	t.Run("uses default buffer size for zero or negative", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](0)
		assert.Equal(t, 100, bus.bufferSize)

		bus = NewGenericEventBus[*TestEvent](-10)
		assert.Equal(t, 100, bus.bufferSize)
	})
}

func TestGenericEventBus_Subscribe(t *testing.T) {
	bus := NewGenericEventBus[*TestEvent](10)

	t.Run("creates subscriber with unique ID", func(t *testing.T) {
		sub1 := bus.Subscribe(nil)
		sub2 := bus.Subscribe(nil)

		assert.NotEmpty(t, sub1.ID)
		assert.NotEmpty(t, sub2.ID)
		assert.NotEqual(t, sub1.ID, sub2.ID)
	})

	t.Run("subscriber count increases", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](10)
		assert.Equal(t, 0, bus.SubscriberCount())

		bus.Subscribe(nil)
		assert.Equal(t, 1, bus.SubscriberCount())

		bus.Subscribe(nil)
		assert.Equal(t, 2, bus.SubscriberCount())
	})
}

func TestGenericEventBus_Unsubscribe(t *testing.T) {
	bus := NewGenericEventBus[*TestEvent](10)

	t.Run("removes subscriber", func(t *testing.T) {
		sub := bus.Subscribe(nil)
		assert.Equal(t, 1, bus.SubscriberCount())

		bus.Unsubscribe(sub.ID)
		assert.Equal(t, 0, bus.SubscriberCount())
	})

	t.Run("closes events channel", func(t *testing.T) {
		sub := bus.Subscribe(nil)
		bus.Unsubscribe(sub.ID)

		// Channel should be closed
		_, ok := <-sub.Events
		assert.False(t, ok)
	})

	t.Run("no-op for unknown ID", func(t *testing.T) {
		// Should not panic
		bus.Unsubscribe("unknown-id")
	})
}

func TestGenericEventBus_Publish(t *testing.T) {
	t.Run("delivers event to all subscribers", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](10)
		sub1 := bus.Subscribe(nil)
		sub2 := bus.Subscribe(nil)

		event := &TestEvent{ID: 1, Message: "test"}
		bus.Publish(event)

		// Both subscribers should receive the event
		select {
		case received := <-sub1.Events:
			assert.Equal(t, event, received)
		case <-time.After(time.Second):
			t.Fatal("sub1 did not receive event")
		}

		select {
		case received := <-sub2.Events:
			assert.Equal(t, event, received)
		case <-time.After(time.Second):
			t.Fatal("sub2 did not receive event")
		}
	})

	t.Run("applies filter function", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](10)

		// Only accept events with Tag == "important"
		filter := func(e *TestEvent) bool {
			return e.Tag == "important"
		}
		sub := bus.Subscribe(filter)

		// Publish non-matching event
		bus.Publish(&TestEvent{ID: 1, Tag: "normal"})

		// Publish matching event
		bus.Publish(&TestEvent{ID: 2, Tag: "important"})

		// Should only receive the matching event
		select {
		case received := <-sub.Events:
			assert.Equal(t, 2, received.ID)
			assert.Equal(t, "important", received.Tag)
		case <-time.After(time.Second):
			t.Fatal("did not receive event")
		}

		// Should not have more events
		select {
		case <-sub.Events:
			t.Fatal("received unexpected event")
		case <-time.After(100 * time.Millisecond):
			// Expected - no more events
		}
	})

	t.Run("drops event when buffer is full", func(t *testing.T) {
		bus := NewGenericEventBus[*TestEvent](2)
		sub := bus.Subscribe(nil)

		// Fill the buffer
		bus.Publish(&TestEvent{ID: 1})
		bus.Publish(&TestEvent{ID: 2})

		// This should be dropped (non-blocking)
		bus.Publish(&TestEvent{ID: 3})

		// Should only receive first two events
		received := make([]*TestEvent, 0)
		for i := 0; i < 2; i++ {
			select {
			case e := <-sub.Events:
				received = append(received, e)
			case <-time.After(time.Second):
				t.Fatal("did not receive expected events")
			}
		}

		assert.Len(t, received, 2)
		assert.Equal(t, 1, received[0].ID)
		assert.Equal(t, 2, received[1].ID)
	})
}

func TestGenericEventBus_Concurrent(t *testing.T) {
	bus := NewGenericEventBus[*TestEvent](1000)

	const numSubscribers = 10
	const numPublishers = 5
	const eventsPerPublisher = 100

	// Create subscribers
	subs := make([]*GenericSubscriber[*TestEvent], numSubscribers)
	for i := 0; i < numSubscribers; i++ {
		subs[i] = bus.Subscribe(nil)
	}

	var wg sync.WaitGroup

	// Start publishers
	for p := 0; p < numPublishers; p++ {
		wg.Add(1)
		go func(publisherID int) {
			defer wg.Done()
			for e := 0; e < eventsPerPublisher; e++ {
				bus.Publish(&TestEvent{
					ID:      publisherID*1000 + e,
					Message: "concurrent test",
				})
			}
		}(p)
	}

	// Collect events from subscribers
	receivedCounts := make([]int, numSubscribers)
	var countsMu sync.Mutex

	for i, sub := range subs {
		wg.Add(1)
		go func(idx int, s *GenericSubscriber[*TestEvent]) {
			defer wg.Done()
			timeout := time.After(5 * time.Second)
			for {
				select {
				case _, ok := <-s.Events:
					if !ok {
						return
					}
					countsMu.Lock()
					receivedCounts[idx]++
					countsMu.Unlock()
				case <-timeout:
					return
				}
			}
		}(i, sub)
	}

	// Wait for publishers to finish
	time.Sleep(100 * time.Millisecond)

	// Unsubscribe all
	for _, sub := range subs {
		bus.Unsubscribe(sub.ID)
	}

	wg.Wait()

	// All subscribers should have received events
	totalExpected := numPublishers * eventsPerPublisher
	for i, count := range receivedCounts {
		// Due to the nature of non-blocking send, we might drop some events
		// But each subscriber should receive at least some events
		assert.Greater(t, count, 0, "subscriber %d received no events", i)
		assert.LessOrEqual(t, count, totalExpected, "subscriber %d received too many events", i)
	}
}
