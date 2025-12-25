// Package subscription provides GraphQL subscription infrastructure.
package subscription_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/subscription"
)

// TestJobProgressBus_Publish tests that the job progress bus can publish events.
func TestJobProgressBus_Publish(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	jobID := uuid.New()
	taskID := uuid.New()
	connectionID := uuid.New()
	startTime := time.Now()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	event := &model.JobProgressEvent{
		JobID:            jobID,
		TaskID:           taskID,
		ConnectionID:     connectionID,
		Status:           model.JobStatusRunning,
		FilesTransferred: 10,
		BytesTransferred: 1024,
		FilesTotal:       100,
		BytesTotal:       10240,
		StartTime:        startTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, jobID, received.JobID)
		assert.Equal(t, taskID, received.TaskID)
		assert.Equal(t, connectionID, received.ConnectionID)
		assert.Equal(t, model.JobStatusRunning, received.Status)
		assert.Equal(t, 10, received.FilesTransferred)
		assert.Equal(t, int64(1024), received.BytesTransferred)
		assert.Equal(t, 100, received.FilesTotal)
		assert.Equal(t, int64(10240), received.BytesTotal)
		assert.Equal(t, startTime, received.StartTime)
		assert.Nil(t, received.EndTime)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for job progress event")
	}
}

// TestJobProgressBus_FilterByTaskID tests filtering by taskID.
func TestJobProgressBus_FilterByTaskID(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	taskID := uuid.New()
	otherTaskID := uuid.New()

	// Subscribe to specific task only
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching task
	matchingEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(matchingEvent)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, taskID, received.TaskID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}

	// Publish event with different task - should not receive
	otherEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           otherTaskID,
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events for different task")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestJobProgressBus_FilterByConnectionID tests filtering by connectionID.
func TestJobProgressBus_FilterByConnectionID(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	connectionID := uuid.New()
	otherConnectionID := uuid.New()

	// Subscribe to specific connection only
	sub := bus.Subscribe(subscription.JobProgressFilter(nil, &connectionID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching connection
	matchingEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     connectionID,
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(matchingEvent)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, connectionID, received.ConnectionID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}

	// Publish event with different connection - should not receive
	otherEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     otherConnectionID,
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events for different connection")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestJobProgressBus_MultipleFilters tests filtering by both taskID and connectionID.
func TestJobProgressBus_MultipleFilters(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	taskID := uuid.New()
	connectionID := uuid.New()

	// Subscribe to specific task and connection
	sub := bus.Subscribe(subscription.JobProgressFilter(&taskID, &connectionID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event matching all criteria
	matchingEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           taskID,
		ConnectionID:     connectionID,
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(matchingEvent)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, taskID, received.TaskID)
		assert.Equal(t, connectionID, received.ConnectionID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}

	// Publish event with different task - should not receive
	wrongTaskEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     connectionID,
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	bus.Publish(wrongTaskEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events with different task")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}

	// Publish event with different connection - should not receive
	wrongConnectionEvent := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           taskID,
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	bus.Publish(wrongConnectionEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events with different connection")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestJobProgressBus_NoFilter tests subscribing without any filter (receive all events).
func TestJobProgressBus_NoFilter(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	// Subscribe without filter (nil filter function)
	sub := bus.Subscribe(subscription.JobProgressFilter(nil, nil))
	defer bus.Unsubscribe(sub.ID)

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, event.JobID, received.JobID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_Unsubscribe tests that unsubscribed channels don't receive events.
func TestJobProgressBus_Unsubscribe(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)

	// Unsubscribe before publishing
	bus.Unsubscribe(sub.ID)

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
	}

	// This should not block or panic
	bus.Publish(event)

	// Channel should be closed after unsubscribe
	select {
	case _, ok := <-sub.Events:
		if ok {
			t.Error("Unsubscribed channel should be closed")
		}
		// Channel is closed, expected
	case <-time.After(100 * time.Millisecond):
		// Expected - no event received (channel closed)
	}
}

// TestJobProgressBus_MultipleSubscribers tests multiple subscribers receive events.
func TestJobProgressBus_MultipleSubscribers(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub1 := bus.Subscribe(nil)
	sub2 := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub1.ID)
	defer bus.Unsubscribe(sub2.ID)

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 5,
		BytesTransferred: 512,
		FilesTotal:       50,
		BytesTotal:       5120,
		StartTime:        time.Now(),
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
			t.Error("Timeout waiting for events")
			return
		}
	}

	assert.True(t, received1)
	assert.True(t, received2)
}

// TestJobProgressBus_SubscriberCount tests the SubscriberCount method.
func TestJobProgressBus_SubscriberCount(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	assert.Equal(t, 0, bus.SubscriberCount())

	sub1 := bus.Subscribe(nil)
	assert.Equal(t, 1, bus.SubscriberCount())

	sub2 := bus.Subscribe(nil)
	assert.Equal(t, 2, bus.SubscriberCount())

	bus.Unsubscribe(sub1.ID)
	assert.Equal(t, 1, bus.SubscriberCount())

	bus.Unsubscribe(sub2.ID)
	assert.Equal(t, 0, bus.SubscriberCount())
}

// TestJobProgressBus_CompletedJob tests event for a completed job.
func TestJobProgressBus_CompletedJob(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusSuccess,
		FilesTransferred: 100,
		BytesTransferred: 10240,
		FilesTotal:       100,
		BytesTotal:       10240,
		StartTime:        startTime,
		EndTime:          &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, model.JobStatusSuccess, received.Status)
		assert.Equal(t, 100, received.FilesTransferred)
		assert.Equal(t, 100, received.FilesTotal)
		require.NotNil(t, received.EndTime)
		assert.Equal(t, endTime.Unix(), received.EndTime.Unix())
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_FailedJob tests event for a failed job.
func TestJobProgressBus_FailedJob(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusFailed,
		FilesTransferred: 50,
		BytesTransferred: 5120,
		FilesTotal:       100,
		BytesTotal:       10240,
		StartTime:        startTime,
		EndTime:          &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, model.JobStatusFailed, received.Status)
		assert.Equal(t, 50, received.FilesTransferred)
		assert.Equal(t, 100, received.FilesTotal)
		require.NotNil(t, received.EndTime)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_PendingJob tests event for a pending job.
func TestJobProgressBus_PendingJob(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusPending,
		FilesTransferred: 0,
		BytesTransferred: 0,
		FilesTotal:       0,
		BytesTotal:       0,
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, model.JobStatusPending, received.Status)
		assert.Equal(t, 0, received.FilesTransferred)
		assert.Equal(t, 0, received.FilesTotal)
		assert.Nil(t, received.EndTime)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_LargeTransfer tests event with large file transfer values.
func TestJobProgressBus_LargeTransfer(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	// Simulate large transfer: 10TB total, 5TB transferred
	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusRunning,
		FilesTransferred: 50000,
		BytesTransferred: 5497558138880, // 5TB
		FilesTotal:       100000,
		BytesTotal:       10995116277760, // 10TB
		StartTime:        time.Now(),
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, 50000, received.FilesTransferred)
		assert.Equal(t, int64(5497558138880), received.BytesTransferred)
		assert.Equal(t, 100000, received.FilesTotal)
		assert.Equal(t, int64(10995116277760), received.BytesTotal)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressBus_CancelledJob tests event for a cancelled job.
func TestJobProgressBus_CancelledJob(t *testing.T) {
	bus := subscription.NewJobProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	startTime := time.Now().Add(-2 * time.Minute)
	endTime := time.Now()

	event := &model.JobProgressEvent{
		JobID:            uuid.New(),
		TaskID:           uuid.New(),
		ConnectionID:     uuid.New(),
		Status:           model.JobStatusCancelled,
		FilesTransferred: 25,
		BytesTransferred: 2560,
		FilesTotal:       100,
		BytesTotal:       10240,
		StartTime:        startTime,
		EndTime:          &endTime,
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, model.JobStatusCancelled, received.Status)
		assert.Equal(t, 25, received.FilesTransferred)
		require.NotNil(t, received.EndTime)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestJobProgressFilter_NilReturnsNil tests that JobProgressFilter returns nil when no filters specified.
func TestJobProgressFilter_NilReturnsNil(t *testing.T) {
	filter := subscription.JobProgressFilter(nil, nil)
	assert.Nil(t, filter)
}

// TestJobProgressFilter_TaskIDOnly tests filter with only taskID specified.
func TestJobProgressFilter_TaskIDOnly(t *testing.T) {
	taskID := uuid.New()
	filter := subscription.JobProgressFilter(&taskID, nil)
	require.NotNil(t, filter)

	// Should accept matching taskID
	matchingEvent := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
	}
	assert.True(t, filter(matchingEvent))

	// Should reject non-matching taskID
	nonMatchingEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
	}
	assert.False(t, filter(nonMatchingEvent))
}

// TestJobProgressFilter_ConnectionIDOnly tests filter with only connectionID specified.
func TestJobProgressFilter_ConnectionIDOnly(t *testing.T) {
	connectionID := uuid.New()
	filter := subscription.JobProgressFilter(nil, &connectionID)
	require.NotNil(t, filter)

	// Should accept matching connectionID
	matchingEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: connectionID,
	}
	assert.True(t, filter(matchingEvent))

	// Should reject non-matching connectionID
	nonMatchingEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
	}
	assert.False(t, filter(nonMatchingEvent))
}

// TestJobProgressFilter_BothFilters tests filter with both taskID and connectionID specified.
func TestJobProgressFilter_BothFilters(t *testing.T) {
	taskID := uuid.New()
	connectionID := uuid.New()
	filter := subscription.JobProgressFilter(&taskID, &connectionID)
	require.NotNil(t, filter)

	// Should accept matching both
	matchingEvent := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: connectionID,
	}
	assert.True(t, filter(matchingEvent))

	// Should reject wrong taskID
	wrongTaskEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: connectionID,
	}
	assert.False(t, filter(wrongTaskEvent))

	// Should reject wrong connectionID
	wrongConnectionEvent := &model.JobProgressEvent{
		TaskID:       taskID,
		ConnectionID: uuid.New(),
	}
	assert.False(t, filter(wrongConnectionEvent))

	// Should reject both wrong
	bothWrongEvent := &model.JobProgressEvent{
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
	}
	assert.False(t, filter(bothWrongEvent))
}
