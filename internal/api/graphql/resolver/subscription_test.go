// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
)

// SubscriptionResolverTestSuite tests Subscription resolvers.
type SubscriptionResolverTestSuite struct {
	ResolverTestSuite
}

func TestSubscriptionResolverSuite(t *testing.T) {
	suite.Run(t, new(SubscriptionResolverTestSuite))
}

// TestJobProgressBus_Publish tests that the job progress bus can publish events.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_Publish() {
	// Get the job progress bus from dependencies
	bus := s.Env.Deps.JobProgressBus

	// Create a subscriber with filter for specific task
	taskID := uuid.New()
	connectionID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	// Publish a progress event
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     connectionID,
		JobID:            uuid.New(),
		Status:           model.JobStatusRunning,
		BytesTransferred: 500,
		FilesTransferred: 10,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	// Wait for the event
	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), taskID, received.TaskID)
		assert.Equal(s.T(), model.JobStatusRunning, received.Status)
		assert.Equal(s.T(), int64(500), received.BytesTransferred)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event")
	}
}

// TestJobProgressBus_MultipleSubscribers tests multiple subscribers receive events.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_MultipleSubscribers() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	// Subscribe both to all events (nil filter)
	sub1 := bus.Subscribe(nil)
	sub2 := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub1.ID)
	defer bus.Unsubscribe(sub2.ID)

	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	// Both subscribers should receive the event
	received1 := false
	received2 := false

	timeout := time.After(time.Second)
	for !received1 || !received2 {
		select {
		case <-sub1.Events:
			received1 = true
		case <-sub2.Events:
			received2 = true
		case <-timeout:
			s.T().Error("Timeout waiting for events")
			return
		}
	}

	assert.True(s.T(), received1)
	assert.True(s.T(), received2)
}

// TestJobProgressBus_Unsubscribe tests that unsubscribed channels don't receive events.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_Unsubscribe() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))

	// Unsubscribe before publishing
	bus.Unsubscribe(sub.ID)

	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	// This should not block or panic
	bus.Publish(event)

	// Channel should be closed after unsubscribe
	select {
	case _, ok := <-sub.Events:
		if ok {
			s.T().Error("Unsubscribed channel should be closed")
		}
		// Channel is closed, expected
	case <-time.After(100 * time.Millisecond):
		// Expected - no event received (channel closed)
	}
}

// TestJobProgressBus_DifferentTasks tests that subscribers with filter only receive matching events.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_DifferentTasks() {
	bus := s.Env.Deps.JobProgressBus

	taskID1 := uuid.New()
	taskID2 := uuid.New()
	// Subscribe to task1 only
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID1, nil))
	defer bus.Unsubscribe(sub.ID)

	// Publish to task2 - sub should not receive it
	event := &model.JobProgressEvent{
		TaskID:       taskID2,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	bus.Publish(event)

	// sub should not receive the event (filtered out)
	select {
	case <-sub.Events:
		s.T().Error("Should not receive events for different task")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event for different task
	}
}

// TestJobProgressBus_CompletedStatus tests that completed status events are published.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_CompletedStatus() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	endTime := time.Now()
	// Publish a completed event
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		JobID:            uuid.New(),
		Status:           model.JobStatusSuccess,
		BytesTransferred: 1000,
		FilesTransferred: 10,
		StartTime:        time.Now().Add(-time.Minute),
		EndTime:          &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), model.JobStatusSuccess, received.Status)
		assert.NotNil(s.T(), received.EndTime)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for completion event")
	}
}

// TestJobProgressBus_FailedStatus tests that failed status events are published.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_FailedStatus() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	endTime := time.Now()
	// Publish a failed event
	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusFailed,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), model.JobStatusFailed, received.Status)
		assert.NotNil(s.T(), received.EndTime)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for failure event")
	}
}

// TestJobProgressBus_AllEventsSubscription tests subscribing to all events (nil filter).
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_AllEventsSubscription() {
	bus := s.Env.Deps.JobProgressBus

	// Subscribe to all events (nil filter)
	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	// Publish to any task
	taskID := uuid.New()
	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	// Should receive the event since no filter
	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), taskID, received.TaskID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}
}

// TestSubscription_JobProgressResolver tests the jobProgress subscription resolver.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgressResolver() {
	// Note: Full WebSocket subscription testing requires a different approach
	// This test verifies the resolver can be instantiated

	// Create a resolver
	resolver := NewResolverForTest(s.Env.Deps)

	// Verify the subscription resolver exists
	assert.NotNil(s.T(), resolver)
}

// TestJobProgressBus_ProgressFields tests that all progress fields are properly set.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_ProgressFields() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now()
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		JobID:            uuid.New(),
		Status:           model.JobStatusRunning,
		BytesTransferred: 5000,
		FilesTransferred: 50,
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), int64(5000), received.BytesTransferred)
		assert.Equal(s.T(), 50, received.FilesTransferred)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event")
	}
}

// TestJobProgressBus_ConnectionFilter tests filtering by connectionID.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_ConnectionFilter() {
	bus := s.Env.Deps.JobProgressBus

	connectionID := uuid.New()
	// Subscribe to specific connection only
	sub := bus.Subscribe(subscription.JobProgressFilter(nil, &connectionID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching connection
	event := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: connectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), connectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}

	// Publish event with different connection - should not receive
	otherEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(), // Different connection
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		s.T().Error("Should not receive events for different connection")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestJobProgressBus_SubscriberCount tests the SubscriberCount method.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_SubscriberCount() {
	bus := s.Env.Deps.JobProgressBus

	initialCount := bus.SubscriberCount()

	sub1 := bus.Subscribe(nil)
	assert.Equal(s.T(), initialCount+1, bus.SubscriberCount())

	sub2 := bus.Subscribe(nil)
	assert.Equal(s.T(), initialCount+2, bus.SubscriberCount())

	bus.Unsubscribe(sub1.ID)
	assert.Equal(s.T(), initialCount+1, bus.SubscriberCount())

	bus.Unsubscribe(sub2.ID)
	assert.Equal(s.T(), initialCount, bus.SubscriberCount())
}

// TestJobProgressBus_CancelledStatus tests that cancelled status events are published.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_CancelledStatus() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	endTime := time.Now()
	// Publish a cancelled event
	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusCancelled,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), model.JobStatusCancelled, received.Status)
		assert.NotNil(s.T(), received.EndTime)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for cancelled event")
	}
}

// TestJobProgressBus_TaskAndConnectionFilter tests filtering by both taskID and connectionID.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_TaskAndConnectionFilter() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	connectionID := uuid.New()
	// Subscribe to specific task AND connection
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, &connectionID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching task AND connection
	event := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: connectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), taskID, received.TaskID)
		assert.Equal(s.T(), connectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}

	// Publish event with matching task but different connection - should not receive
	wrongConnEvent := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(), // Different connection
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	bus.Publish(wrongConnEvent)

	select {
	case <-sub.Events:
		s.T().Error("Should not receive events with different connection")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	// Publish event with different task but matching connection - should not receive
	wrongTaskEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(), // Different task
		ConnectionID: connectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	bus.Publish(wrongTaskEvent)

	select {
	case <-sub.Events:
		s.T().Error("Should not receive events with different task")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}
