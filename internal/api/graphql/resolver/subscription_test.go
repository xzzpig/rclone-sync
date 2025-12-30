// Package resolver provides GraphQL resolver tests.
package resolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/resolver"
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
	res := NewResolverForTest(s.Env.Deps)

	// Verify the subscription resolver exists
	assert.NotNil(s.T(), res)
}

// TestSubscription_JobProgress_NilBus tests that JobProgress returns closed channel when bus is nil.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_NilBus() {
	// Create dependencies with nil JobProgressBus
	deps := &resolver.Dependencies{
		JobProgressBus:      nil,
		TransferProgressBus: nil,
	}
	res := resolver.New(deps)

	ctx := context.Background()
	ch, err := res.Subscription().JobProgress(ctx, nil, nil)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Channel should be closed immediately
	select {
	case _, ok := <-ch:
		assert.False(s.T(), ok, "Channel should be closed when bus is nil")
	case <-time.After(100 * time.Millisecond):
		s.T().Error("Channel should be closed immediately when bus is nil")
	}
}

// TestSubscription_JobProgress_ReceivesEvents tests that JobProgress resolver receives events from the bus.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_ReceivesEvents() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := res.Subscription().JobProgress(ctx, nil, nil)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Publish an event
	taskID := uuid.New()
	connectionID := uuid.New()
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     connectionID,
		JobID:            uuid.New(),
		Status:           model.JobStatusRunning,
		BytesTransferred: 1000,
		FilesTransferred: 5,
		StartTime:        time.Now(),
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Env.Deps.JobProgressBus.Publish(event)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), taskID, received.TaskID)
		assert.Equal(s.T(), connectionID, received.ConnectionID)
		assert.Equal(s.T(), model.JobStatusRunning, received.Status)
		assert.Equal(s.T(), int64(1000), received.BytesTransferred)
		assert.Equal(s.T(), 5, received.FilesTransferred)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event from JobProgress resolver")
	}
}

// TestSubscription_JobProgress_ContextCancellation tests that JobProgress cleans up when context is cancelled.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_ContextCancellation() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())

	initialCount := s.Env.Deps.JobProgressBus.SubscriberCount()

	ch, err := res.Subscription().JobProgress(ctx, nil, nil)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Wait a bit for subscription to be established
	time.Sleep(50 * time.Millisecond)
	assert.Equal(s.T(), initialCount+1, s.Env.Deps.JobProgressBus.SubscriberCount())

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Subscriber should be removed
	assert.Equal(s.T(), initialCount, s.Env.Deps.JobProgressBus.SubscriberCount())

	// Channel should be closed
	select {
	case _, ok := <-ch:
		assert.False(s.T(), ok, "Channel should be closed after context cancellation")
	case <-time.After(100 * time.Millisecond):
		s.T().Error("Channel should be closed after context cancellation")
	}
}

// TestSubscription_JobProgress_WithTaskFilter tests that JobProgress filters by taskID.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_WithTaskFilter() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetTaskID := uuid.New()
	ch, err := res.Subscription().JobProgress(ctx, &targetTaskID, nil)
	assert.NoError(s.T(), err)

	// Publish event with matching taskID
	matchingEvent := &model.JobProgressEvent{
		TaskID:       targetTaskID,
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Env.Deps.JobProgressBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetTaskID, received.TaskID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with different taskID - should not be received
	otherEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(), // Different taskID
		ConnectionID: uuid.New(),
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	s.Env.Deps.JobProgressBus.Publish(otherEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events for different taskID")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event for different task
	}
}

// TestSubscription_JobProgress_WithConnectionFilter tests that JobProgress filters by connectionID.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_WithConnectionFilter() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetConnectionID := uuid.New()
	ch, err := res.Subscription().JobProgress(ctx, nil, &targetConnectionID)
	assert.NoError(s.T(), err)

	// Publish event with matching connectionID
	matchingEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: targetConnectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Env.Deps.JobProgressBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetConnectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with different connectionID - should not be received
	otherEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(), // Different connectionID
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	s.Env.Deps.JobProgressBus.Publish(otherEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events for different connectionID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestSubscription_JobProgress_WithBothFilters tests that JobProgress filters by both taskID and connectionID.
func (s *SubscriptionResolverTestSuite) TestSubscription_JobProgress_WithBothFilters() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetTaskID := uuid.New()
	targetConnectionID := uuid.New()
	ch, err := res.Subscription().JobProgress(ctx, &targetTaskID, &targetConnectionID)
	assert.NoError(s.T(), err)

	// Publish event with matching both
	matchingEvent := &model.JobProgressEvent{
		TaskID:       targetTaskID,
		ConnectionID: targetConnectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Env.Deps.JobProgressBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetTaskID, received.TaskID)
		assert.Equal(s.T(), targetConnectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with matching taskID but different connectionID - should not be received
	wrongConnEvent := &model.JobProgressEvent{
		TaskID:       targetTaskID,
		ConnectionID: uuid.New(), // Different connectionID
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	s.Env.Deps.JobProgressBus.Publish(wrongConnEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events with different connectionID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	// Publish event with matching connectionID but different taskID - should not be received
	wrongTaskEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(), // Different taskID
		ConnectionID: targetConnectionID,
		JobID:        uuid.New(),
		Status:       model.JobStatusRunning,
		StartTime:    time.Now(),
	}

	s.Env.Deps.JobProgressBus.Publish(wrongTaskEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events with different taskID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestSubscription_TransferProgress_NilBus tests that TransferProgress returns closed channel when bus is nil.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_NilBus() {
	// Create dependencies with nil TransferProgressBus
	deps := &resolver.Dependencies{
		JobProgressBus:      nil,
		TransferProgressBus: nil,
	}
	res := resolver.New(deps)

	ctx := context.Background()
	ch, err := res.Subscription().TransferProgress(ctx, nil, nil, nil)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Channel should be closed immediately
	select {
	case _, ok := <-ch:
		assert.False(s.T(), ok, "Channel should be closed when bus is nil")
	case <-time.After(100 * time.Millisecond):
		s.T().Error("Channel should be closed immediately when bus is nil")
	}
}

// TestSubscription_TransferProgress_ReceivesEvents tests that TransferProgress resolver receives events.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_ReceivesEvents() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := res.Subscription().TransferProgress(ctx, nil, nil, nil)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Publish an event
	jobID := uuid.New()
	taskID := uuid.New()
	connectionID := uuid.New()
	event := &model.TransferProgressEvent{
		JobID:        jobID,
		TaskID:       taskID,
		ConnectionID: connectionID,
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		s.Env.Deps.TransferProgressBus.Publish(event)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), jobID, received.JobID)
		assert.Equal(s.T(), taskID, received.TaskID)
		assert.Equal(s.T(), connectionID, received.ConnectionID)
		assert.Len(s.T(), received.Transfers, 1)
		assert.Equal(s.T(), "file1.txt", received.Transfers[0].Name)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event from TransferProgress resolver")
	}
}

// TestSubscription_TransferProgress_ContextCancellation tests cleanup on context cancellation.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_ContextCancellation() {
	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())

	initialCount := s.Env.Deps.TransferProgressBus.SubscriberCount()

	ch, err := res.Subscription().TransferProgress(ctx, nil, nil, nil)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), ch)

	// Wait a bit for subscription to be established
	time.Sleep(50 * time.Millisecond)
	assert.Equal(s.T(), initialCount+1, s.Env.Deps.TransferProgressBus.SubscriberCount())

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Subscriber should be removed
	assert.Equal(s.T(), initialCount, s.Env.Deps.TransferProgressBus.SubscriberCount())

	// Channel should be closed
	select {
	case _, ok := <-ch:
		assert.False(s.T(), ok, "Channel should be closed after context cancellation")
	case <-time.After(100 * time.Millisecond):
		s.T().Error("Channel should be closed after context cancellation")
	}
}

// TestSubscription_TransferProgress_WithJobFilter tests that TransferProgress filters by jobID.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_WithJobFilter() {
	// Create a TransferProgressBus for this test
	transferBus := subscription.NewTransferProgressBus()
	s.Env.Deps.TransferProgressBus = transferBus

	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetJobID := uuid.New()
	ch, err := res.Subscription().TransferProgress(ctx, nil, nil, &targetJobID)
	assert.NoError(s.T(), err)

	// Publish event with matching jobID
	matchingEvent := &model.TransferProgressEvent{
		JobID:        targetJobID,
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		transferBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetJobID, received.JobID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with different jobID - should not be received
	otherEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(), // Different jobID
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	transferBus.Publish(otherEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events for different jobID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestSubscription_TransferProgress_WithTaskFilter tests that TransferProgress filters by taskID.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_WithTaskFilter() {
	// Create a TransferProgressBus for this test
	transferBus := subscription.NewTransferProgressBus()
	s.Env.Deps.TransferProgressBus = transferBus

	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetTaskID := uuid.New()
	ch, err := res.Subscription().TransferProgress(ctx, nil, &targetTaskID, nil)
	assert.NoError(s.T(), err)

	// Publish event with matching taskID
	matchingEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       targetTaskID,
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		transferBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetTaskID, received.TaskID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with different taskID - should not be received
	otherEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(), // Different taskID
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	transferBus.Publish(otherEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events for different taskID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestSubscription_TransferProgress_WithConnectionFilter tests that TransferProgress filters by connectionID.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_WithConnectionFilter() {
	// Create a TransferProgressBus for this test
	transferBus := subscription.NewTransferProgressBus()
	s.Env.Deps.TransferProgressBus = transferBus

	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetConnectionID := uuid.New()
	ch, err := res.Subscription().TransferProgress(ctx, &targetConnectionID, nil, nil)
	assert.NoError(s.T(), err)

	// Publish event with matching connectionID
	matchingEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: targetConnectionID,
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		transferBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetConnectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with different connectionID - should not be received
	otherEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(), // Different connectionID
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	transferBus.Publish(otherEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events for different connectionID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestSubscription_TransferProgress_WithAllFilters tests filtering by all parameters.
func (s *SubscriptionResolverTestSuite) TestSubscription_TransferProgress_WithAllFilters() {
	// Create a TransferProgressBus for this test
	transferBus := subscription.NewTransferProgressBus()
	s.Env.Deps.TransferProgressBus = transferBus

	res := NewResolverForTest(s.Env.Deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	targetConnectionID := uuid.New()
	targetTaskID := uuid.New()
	targetJobID := uuid.New()
	ch, err := res.Subscription().TransferProgress(ctx, &targetConnectionID, &targetTaskID, &targetJobID)
	assert.NoError(s.T(), err)

	// Publish event with all matching
	matchingEvent := &model.TransferProgressEvent{
		JobID:        targetJobID,
		TaskID:       targetTaskID,
		ConnectionID: targetConnectionID,
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		transferBus.Publish(matchingEvent)
	}()

	select {
	case received := <-ch:
		assert.Equal(s.T(), targetJobID, received.JobID)
		assert.Equal(s.T(), targetTaskID, received.TaskID)
		assert.Equal(s.T(), targetConnectionID, received.ConnectionID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for filtered event")
	}

	// Publish event with one mismatching field - should not be received
	wrongJobEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(), // Different jobID
		TaskID:       targetTaskID,
		ConnectionID: targetConnectionID,
		Transfers:    []*model.TransferItem{{Name: "file2.txt", Size: 1000, Bytes: 500}},
	}

	transferBus.Publish(wrongJobEvent)

	select {
	case <-ch:
		s.T().Error("Should not receive events with different jobID")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
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

// TestJobProgressBus_TotalFields tests that FilesTotal and BytesTotal fields are properly set.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_TotalFields() {
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
		FilesTransferred: 45,
		BytesTransferred: 12000,
		FilesTotal:       128,
		BytesTotal:       10485760, // 10 MB
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), 45, received.FilesTransferred)
		assert.Equal(s.T(), int64(12000), received.BytesTransferred)
		assert.Equal(s.T(), 128, received.FilesTotal)
		assert.Equal(s.T(), int64(10485760), received.BytesTotal)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event with total fields")
	}
}

// TestJobProgressBus_ZeroTotalFields tests that zero values for total fields are handled correctly.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_ZeroTotalFields() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now()
	// Event with zero totals (scanning just started, no totals yet)
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		JobID:            uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 0,
		BytesTransferred: 0,
		FilesTotal:       0,
		BytesTotal:       0,
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), 0, received.FilesTotal)
		assert.Equal(s.T(), int64(0), received.BytesTotal)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event with zero totals")
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

// TestTransferProgressBus_Publish tests that the transfer progress bus can publish events.
func (s *SubscriptionResolverTestSuite) TestTransferProgressBus_Publish() {
	// Create a new bus for this test
	bus := subscription.NewTransferProgressBus()

	jobID := uuid.New()
	taskID := uuid.New()
	connectionID := uuid.New()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	event := &model.TransferProgressEvent{
		JobID:        jobID,
		TaskID:       taskID,
		ConnectionID: connectionID,
		Transfers: []*model.TransferItem{
			{
				Name:  "file1.txt",
				Size:  1000,
				Bytes: 500,
			},
		},
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), jobID, received.JobID)
		assert.Equal(s.T(), taskID, received.TaskID)
		assert.Equal(s.T(), connectionID, received.ConnectionID)
		assert.Len(s.T(), received.Transfers, 1)
		assert.Equal(s.T(), "file1.txt", received.Transfers[0].Name)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for transfer progress event")
	}
}

// TestTransferProgressBus_FilterByJobID tests filtering by jobID.
func (s *SubscriptionResolverTestSuite) TestTransferProgressBus_FilterByJobID() {
	bus := subscription.NewTransferProgressBus()

	jobID := uuid.New()
	otherJobID := uuid.New()

	// Subscribe to specific job only
	sub := bus.Subscribe(subscription.TransferProgressFilter(nil, nil, &jobID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching job
	matchingEvent := &model.TransferProgressEvent{
		JobID:        jobID,
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		bus.Publish(matchingEvent)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), jobID, received.JobID)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}

	// Publish event with different job - should not receive
	otherEvent := &model.TransferProgressEvent{
		JobID:        otherJobID,
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		s.T().Error("Should not receive events for different job")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestTransferProgressBus_EmptyTransfers tests events with empty transfer list.
func (s *SubscriptionResolverTestSuite) TestTransferProgressBus_EmptyTransfers() {
	bus := subscription.NewTransferProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	// Empty transfers list indicates all transfers completed
	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers:    []*model.TransferItem{},
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Empty(s.T(), received.Transfers)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}
}

// TestTransferProgressBus_CompletedTransfer tests event with a completed transfer (bytes == size).
func (s *SubscriptionResolverTestSuite) TestTransferProgressBus_CompletedTransfer() {
	bus := subscription.NewTransferProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "completed.txt", Size: 1000, Bytes: 1000}, // Completed: bytes == size
		},
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Len(s.T(), received.Transfers, 1)
		assert.Equal(s.T(), int64(1000), received.Transfers[0].Size)
		assert.Equal(s.T(), int64(1000), received.Transfers[0].Bytes)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_FilesDeletedAndErrorCount tests that FilesDeleted and ErrorCount fields are properly set.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_FilesDeletedAndErrorCount() {
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
		FilesTransferred: 45,
		BytesTransferred: 12000,
		FilesTotal:       128,
		BytesTotal:       10485760,
		FilesDeleted:     15, // New field: deleted files count
		ErrorCount:       3,  // New field: error count
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), 45, received.FilesTransferred)
		assert.Equal(s.T(), int64(12000), received.BytesTransferred)
		assert.Equal(s.T(), 15, received.FilesDeleted)
		assert.Equal(s.T(), 3, received.ErrorCount)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event with FilesDeleted and ErrorCount")
	}
}

// TestJobProgressBus_ZeroFilesDeletedAndErrorCount tests zero values for FilesDeleted and ErrorCount.
func (s *SubscriptionResolverTestSuite) TestJobProgressBus_ZeroFilesDeletedAndErrorCount() {
	bus := s.Env.Deps.JobProgressBus

	taskID := uuid.New()
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now()
	// Event with zero deletes and errors
	event := &model.JobProgressEvent{
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		JobID:            uuid.New(),
		Status:           model.JobStatusSuccess,
		FilesTransferred: 10,
		BytesTransferred: 1024,
		FilesDeleted:     0,
		ErrorCount:       0,
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(s.T(), 0, received.FilesDeleted)
		assert.Equal(s.T(), 0, received.ErrorCount)
	case <-time.After(time.Second):
		s.T().Error("Timeout waiting for progress event with zero FilesDeleted and ErrorCount")
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
