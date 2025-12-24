package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"github.com/xzzpig/rclone-sync/internal/core/scheduler"
)

// MockRunner is a mock for the Runner interface
type MockRunner struct {
	mock.Mock
}

func (m *MockRunner) Start() { m.Called() }
func (m *MockRunner) Stop()  { m.Called() }
func (m *MockRunner) StartTask(task *ent.Task, trigger model.JobTrigger) error {
	args := m.Called(task, string(trigger))
	return args.Error(0)
}
func (m *MockRunner) StopTask(taskID uuid.UUID) error {
	args := m.Called(taskID)
	return args.Error(0)
}
func (m *MockRunner) IsRunning(taskID uuid.UUID) bool {
	args := m.Called(taskID)
	return args.Bool(0)
}

// MockTaskService is a mock for the TaskService interface
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) GetTask(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ent.Task), args.Error(1)
}

func (m *MockTaskService) GetTaskWithConnection(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ent.Task), args.Error(1)
}

func (m *MockTaskService) ListAllTasks(ctx context.Context) ([]*ent.Task, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*ent.Task), args.Error(1)
}

func setupTest(t *testing.T) {
	t.Helper()
	logger.InitLogger(logger.EnvironmentDevelopment, logger.LogLevelDebug)
}

func TestScheduler_Start_LoadsScheduledTasks(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	task1 := &ent.Task{ID: uuid.New(), Name: "Scheduled Task", Schedule: "* * * * * *"}
	task2 := &ent.Task{ID: uuid.New(), Name: "Unscheduled Task", Schedule: ""}
	tasks := []*ent.Task{task1, task2}

	mockTaskSvc.On("ListAllTasks", mock.Anything).Return(tasks, nil)
	// When cron triggers, scheduler will reload the task from DB using GetTaskWithConnection
	mockTaskSvc.On("GetTaskWithConnection", mock.Anything, task1.ID).Return(task1, nil)
	// We expect StartTask to be called for the scheduled task.
	// We use a WaitGroup or channel to handle the async nature of cron.
	startedChan := make(chan bool, 1)
	mockRunner.On("StartTask", task1, string(model.JobTriggerSchedule)).Return(nil).Run(func(args mock.Arguments) {
		startedChan <- true
	})

	s := scheduler.NewScheduler(mockTaskSvc, mockRunner, cron.WithSeconds())
	s.Start()
	defer s.Stop()

	// Wait for the cron job to be triggered
	select {
	case <-startedChan:
		// Success
	case <-time.After(1500 * time.Millisecond): // Cron ticks every second
		t.Fatal("timed out waiting for scheduled task to start")
	}

	mockTaskSvc.AssertExpectations(t)
	mockRunner.AssertExpectations(t)
}

func TestScheduler_AddTask_And_RemoveTask(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	task := &ent.Task{ID: uuid.New(), Name: "Dynamic Task", Schedule: "* * * * * *"}

	// The scheduler calls ListAllTasks on Start, so we need to expect that.
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil).Once()

	s := scheduler.NewScheduler(mockTaskSvc, mockRunner, cron.WithSeconds())
	s.Start()
	defer s.Stop()

	// Add the task
	err := s.AddTask(task)
	assert.NoError(t, err)

	// When cron triggers, scheduler will reload the task from DB using GetTaskWithConnection
	mockTaskSvc.On("GetTaskWithConnection", mock.Anything, task.ID).Return(task, nil)
	// Expect it to run
	startedChan := make(chan bool, 1)
	mockRunner.On("StartTask", task, string(model.JobTriggerSchedule)).Return(nil).Run(func(args mock.Arguments) {
		startedChan <- true
	})

	select {
	case <-startedChan:
		// Success
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("timed out waiting for added task to start")
	}

	// Remove the task
	err = s.RemoveTask(task)
	assert.NoError(t, err)

	// To verify removal, we check that StartTask is not called again.
	// We can wait for a period longer than the cron interval and assert that the mock
	// was only called once in total.
	time.Sleep(1500 * time.Millisecond)

	mockRunner.AssertNumberOfCalls(t, "StartTask", 1)
	mockTaskSvc.AssertExpectations(t)
}

func TestScheduler_StartStopIdempotency(t *testing.T) {
	setupTest(t)
	mockTaskSvc := new(MockTaskService)
	mockRunner := new(MockRunner)

	// Expect ListAllTasks to be called only once initially
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil).Once()

	s := scheduler.NewScheduler(mockTaskSvc, mockRunner)

	// 1. Test Start idempotency
	s.Start()
	s.Start() // Second call should be a no-op

	mockTaskSvc.AssertExpectations(t)

	// 2. Test Stop idempotency
	s.Stop()
	s.Stop() // Second call should be a no-op

	// 3. Test restart capability
	// After stopping, we should be able to start again.
	// This will trigger ListAllTasks again.
	mockTaskSvc.On("ListAllTasks", mock.Anything).Return([]*ent.Task{}, nil).Once()
	s.Start()
	mockTaskSvc.AssertExpectations(t)

	s.Stop() // Final cleanup
}
