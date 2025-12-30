package services

import (
	"context"
	"errors"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
)

// TaskService provides operations for managing sync tasks.
type TaskService struct {
	client *ent.Client
}

// NewTaskService creates a new TaskService instance.
func NewTaskService(client *ent.Client) *TaskService {
	return &TaskService{client: client}
}

// withLatestJobPredicate 返回一个 JobQuery 的过滤器,用于只查询每个 task 的最新 job
// 通过子查询来实现:对于每个 task,只选择 start_time 最大的 job
func withLatestJobPredicate(q *ent.JobQuery) {
	q.Where(func(s *sql.Selector) {
		// 创建子查询,获取每个 task 的最新 job ID
		// 子查询: SELECT id FROM jobs j2 WHERE j2.task_jobs = jobs.task_jobs ORDER BY start_time DESC LIMIT 1
		subquery := sql.Select("id").
			From(sql.Table(job.Table).As("j2")).
			Where(sql.ColumnsEQ(
				sql.Table("j2").C(job.TaskColumn),
				s.C(job.TaskColumn),
			)).
			OrderBy(sql.Desc(sql.Table("j2").C(job.FieldStartTime))).
			Limit(1)

		// 只选择 ID 在子查询结果中的 job
		s.Where(sql.In(s.C(job.FieldID), subquery))
	})
}

// CreateTask creates a new sync task with the given parameters.
func (s *TaskService) CreateTask(ctx context.Context, name, sourcePath string, connectionID uuid.UUID, remotePath, direction, schedule string, realtime bool, options *model.TaskSyncOptions) (*ent.Task, error) {
	t, err := s.client.Task.Create().
		SetName(name).
		SetSourcePath(sourcePath).
		SetConnectionID(connectionID).
		SetRemotePath(remotePath).
		SetDirection(model.SyncDirection(direction)).
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

// ListAllTasks retrieves all tasks with their latest job and connection.
func (s *TaskService) ListAllTasks(ctx context.Context) ([]*ent.Task, error) {
	tasks, err := s.client.Task.Query().
		WithJobs(withLatestJobPredicate).
		WithConnection().
		All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return tasks, nil
}

// ListTasksByConnection retrieves tasks by connection ID with their latest job.
func (s *TaskService) ListTasksByConnection(ctx context.Context, connectionID uuid.UUID) ([]*ent.Task, error) {
	query := s.client.Task.Query()
	if connectionID != uuid.Nil {
		query = query.Where(task.ConnectionIDEQ(connectionID))
	}

	// 使用 WithJobs 配置子查询来获取每个 task 的最新 job
	tasks, err := query.WithJobs(withLatestJobPredicate).WithConnection().All(ctx)

	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return tasks, nil
}

// GetTask retrieves a task by ID.
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

// GetTaskWithConnection retrieves a task by ID with its connection.
func (s *TaskService) GetTaskWithConnection(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	t, err := s.client.Task.Query().
		Where(task.IDEQ(id)).
		WithConnection().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return t, nil
}

// GetTaskWithJobs retrieves a task by ID with its latest job.
func (s *TaskService) GetTaskWithJobs(ctx context.Context, id uuid.UUID) (*ent.Task, error) {
	t, err := s.client.Task.Query().
		Where(task.IDEQ(id)).
		WithJobs(withLatestJobPredicate).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return t, nil
}

// UpdateTask updates an existing task with the given parameters.
func (s *TaskService) UpdateTask(ctx context.Context, id uuid.UUID, name, sourcePath string, connectionID uuid.UUID, remotePath, direction, schedule string, realtime bool, options *model.TaskSyncOptions) (*ent.Task, error) {
	t, err := s.client.Task.UpdateOneID(id).
		SetName(name).
		SetSourcePath(sourcePath).
		SetConnectionID(connectionID).
		SetRemotePath(remotePath).
		SetDirection(model.SyncDirection(direction)).
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

// DeleteTask deletes a task by ID.
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

// ListTasksPaginated lists tasks with pagination.
func (s *TaskService) ListTasksPaginated(ctx context.Context, limit, offset int) ([]*ent.Task, int, error) {
	query := s.client.Task.Query().
		Order(ent.Desc(task.FieldCreatedAt))

	// Get total count
	totalCount, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	// Apply pagination and fetch items
	tasks, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	return tasks, totalCount, nil
}

// ListTasksByConnectionPaginated lists tasks by connection ID with pagination.
func (s *TaskService) ListTasksByConnectionPaginated(ctx context.Context, connectionID uuid.UUID, limit, offset int) ([]*ent.Task, int, error) {
	query := s.client.Task.Query().
		Where(task.ConnectionID(connectionID)).
		Order(ent.Desc(task.FieldCreatedAt))

	// Get total count
	totalCount, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	// Apply pagination and fetch items
	tasks, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	return tasks, totalCount, nil
}

// ListJobsByTaskPaginated lists jobs for a task with pagination.
func (s *TaskService) ListJobsByTaskPaginated(ctx context.Context, taskID uuid.UUID, limit, offset int) ([]*ent.Job, int, error) {
	query := s.client.Job.Query().
		Where(job.HasTaskWith(task.ID(taskID))).
		Order(ent.Desc(job.FieldStartTime))

	// Get total count
	totalCount, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	// Apply pagination and fetch items
	jobs, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, errors.Join(errs.ErrSystem, err)
	}

	return jobs, totalCount, nil
}

var _ ports.TaskService = (*TaskService)(nil)
