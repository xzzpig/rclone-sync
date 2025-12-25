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

// TestTransferProgressBus_Publish tests that the transfer progress bus can publish events.
func TestTransferProgressBus_Publish(t *testing.T) {
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
		assert.Equal(t, jobID, received.JobID)
		assert.Equal(t, taskID, received.TaskID)
		assert.Equal(t, connectionID, received.ConnectionID)
		require.Len(t, received.Transfers, 1)
		assert.Equal(t, "file1.txt", received.Transfers[0].Name)
		assert.Equal(t, int64(1000), received.Transfers[0].Size)
		assert.Equal(t, int64(500), received.Transfers[0].Bytes)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for transfer progress event")
	}
}

// TestTransferProgressBus_MultipleTransfers tests events with multiple transfer items.
func TestTransferProgressBus_MultipleTransfers(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
			{Name: "file2.pdf", Size: 2000, Bytes: 1500},
			{Name: "folder/file3.doc", Size: 3000, Bytes: 3000}, // Completed
		},
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		require.Len(t, received.Transfers, 3)
		assert.Equal(t, "file1.txt", received.Transfers[0].Name)
		assert.Equal(t, "file2.pdf", received.Transfers[1].Name)
		assert.Equal(t, "folder/file3.doc", received.Transfers[2].Name)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for transfer progress event")
	}
}

// TestTransferProgressBus_FilterByConnectionID tests filtering by connectionID.
func TestTransferProgressBus_FilterByConnectionID(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	connectionID := uuid.New()
	otherConnectionID := uuid.New()

	// Subscribe to specific connection only
	sub := bus.Subscribe(subscription.TransferProgressFilter(&connectionID, nil, nil))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching connection
	matchingEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: connectionID,
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
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
	otherEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: otherConnectionID,
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events for different connection")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestTransferProgressBus_FilterByTaskID tests filtering by taskID.
func TestTransferProgressBus_FilterByTaskID(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	taskID := uuid.New()
	otherTaskID := uuid.New()

	// Subscribe to specific task only
	sub := bus.Subscribe(subscription.TransferProgressFilter(nil, &taskID, nil))
	defer bus.Unsubscribe(sub.ID)

	// Publish event with matching task
	matchingEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       taskID,
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
		assert.Equal(t, taskID, received.TaskID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}

	// Publish event with different task - should not receive
	otherEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       otherTaskID,
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	bus.Publish(otherEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events for different task")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestTransferProgressBus_FilterByJobID tests filtering by jobID.
func TestTransferProgressBus_FilterByJobID(t *testing.T) {
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
		assert.Equal(t, jobID, received.JobID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
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
		t.Error("Should not receive events for different job")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestTransferProgressBus_MultipleFilters tests filtering by multiple criteria.
func TestTransferProgressBus_MultipleFilters(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	connectionID := uuid.New()
	taskID := uuid.New()
	jobID := uuid.New()

	// Subscribe to specific connection, task, and job
	sub := bus.Subscribe(subscription.TransferProgressFilter(&connectionID, &taskID, &jobID))
	defer bus.Unsubscribe(sub.ID)

	// Publish event matching all criteria
	matchingEvent := &model.TransferProgressEvent{
		JobID:        jobID,
		TaskID:       taskID,
		ConnectionID: connectionID,
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
	}

	go func() {
		bus.Publish(matchingEvent)
	}()

	select {
	case received := <-sub.Events:
		assert.Equal(t, jobID, received.JobID)
		assert.Equal(t, taskID, received.TaskID)
		assert.Equal(t, connectionID, received.ConnectionID)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}

	// Publish event with different job - should not receive
	wrongJobEvent := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       taskID,
		ConnectionID: connectionID,
		Transfers: []*model.TransferItem{
			{Name: "file2.txt", Size: 1000, Bytes: 500},
		},
	}

	bus.Publish(wrongJobEvent)

	select {
	case <-sub.Events:
		t.Error("Should not receive events with different job")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

// TestTransferProgressBus_Unsubscribe tests that unsubscribed channels don't receive events.
func TestTransferProgressBus_Unsubscribe(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	sub := bus.Subscribe(nil)

	// Unsubscribe before publishing
	bus.Unsubscribe(sub.ID)

	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers:    []*model.TransferItem{},
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

// TestTransferProgressBus_MultipleSubscribers tests multiple subscribers receive events.
func TestTransferProgressBus_MultipleSubscribers(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	sub1 := bus.Subscribe(nil)
	sub2 := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub1.ID)
	defer bus.Unsubscribe(sub2.ID)

	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "file1.txt", Size: 1000, Bytes: 500},
		},
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

// TestTransferProgressBus_EmptyTransfers tests events with empty transfer list.
func TestTransferProgressBus_EmptyTransfers(t *testing.T) {
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
		assert.Empty(t, received.Transfers)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestTransferProgressBus_SubscriberCount tests the SubscriberCount method.
func TestTransferProgressBus_SubscriberCount(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

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

// TestTransferProgressBus_CompletedTransfer tests event with a completed transfer (bytes == size).
func TestTransferProgressBus_CompletedTransfer(t *testing.T) {
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
		require.Len(t, received.Transfers, 1)
		assert.Equal(t, int64(1000), received.Transfers[0].Size)
		assert.Equal(t, int64(1000), received.Transfers[0].Bytes)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// TestTransferProgressBus_LargeFileTransfer tests event with large file transfer.
func TestTransferProgressBus_LargeFileTransfer(t *testing.T) {
	bus := subscription.NewTransferProgressBus()

	sub := bus.Subscribe(nil)
	defer bus.Unsubscribe(sub.ID)

	// 10GB file
	event := &model.TransferProgressEvent{
		JobID:        uuid.New(),
		TaskID:       uuid.New(),
		ConnectionID: uuid.New(),
		Transfers: []*model.TransferItem{
			{Name: "large_video.mp4", Size: 10737418240, Bytes: 5368709120}, // 5GB/10GB
		},
	}

	go func() {
		bus.Publish(event)
	}()

	select {
	case received := <-sub.Events:
		require.Len(t, received.Transfers, 1)
		assert.Equal(t, int64(10737418240), received.Transfers[0].Size)
		assert.Equal(t, int64(5368709120), received.Transfers[0].Bytes)
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}
