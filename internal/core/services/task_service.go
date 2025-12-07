package services

import (
	"context"
	"errors"

	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"

	"github.com/google/uuid"
)

type TaskService struct {
	client *ent.Client
}

func NewTaskService(client *ent.Client) *TaskService {
	return &TaskService{client: client}
}

func (s *TaskService) CreateTask(ctx context.Context, name, sourcePath, remoteName, remotePath, direction, schedule string, realtime bool, options map[string]interface{}) (*ent.Task, error) {
	t, err := s.client.Task.Create().
		SetName(name).
		SetSourcePath(sourcePath).
		SetRemoteName(remoteName).
		SetRemotePath(remotePath).
		SetDirection(task.Direction(direction)).
		SetSchedule(schedule).
		SetRealtime(realtime).
		SetOptions(options).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, errors.Join(errs.ErrAlreadyExists, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return t, nil
}

func (s *TaskService) ListAllTasks(ctx context.Context) ([]*ent.Task, error) {
	tasks, err := s.client.Task.Query().All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return tasks, nil
}

func (s *TaskService) GetTask(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	t, err := s.client.Task.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return t, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, name, sourcePath, remoteName, remotePath, direction, schedule string, realtime bool, options map[string]interface{}) (*ent.Task, error) {
	t, err := s.client.Task.UpdateOneID(id).
		SetName(name).
		SetSourcePath(sourcePath).
		SetRemoteName(remoteName).
		SetRemotePath(remotePath).
		SetDirection(task.Direction(direction)).
		SetSchedule(schedule).
		SetRealtime(realtime).
		SetOptions(options).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		if ent.IsConstraintError(err) {
			return nil, errors.Join(errs.ErrAlreadyExists, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return t, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id uuid.UUID) error {
	err := s.client.Task.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.Join(errs.ErrNotFound, err)
		}
		return errors.Join(errs.ErrSystem, err)
	}
	return nil
}
