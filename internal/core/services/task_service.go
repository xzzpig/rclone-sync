package services

import (
	"context"
	"errors"

	"entgo.io/ent/dialect/sql"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/job"
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

// withLatestJobPredicate 返回一个 JobQuery 的过滤器,用于只查询每个 task 的最新 job
// 通过子查询来实现:对于每个 task,只选择 start_time 最大的 job
func withLatestJobPredicate(q *ent.JobQuery) {
	q.Where(func(s *sql.Selector) {
		// 创建子查询,获取每个 task 的最新 job ID
		// 子查询: SELECT id FROM jobs j2 WHERE j2.task_jobs = jobs.task_jobs ORDER BY start_time DESC LIMIT 1
		subquery := sql.Select("id").
			From(sql.Table("jobs").As("j2")).
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
	tasks, err := s.client.Task.Query().
		WithJobs(withLatestJobPredicate).
		All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return tasks, nil
}

func (s *TaskService) ListTasksByRemote(ctx context.Context, remoteName string) ([]*ent.Task, error) {
	query := s.client.Task.Query()
	if remoteName != "" {
		query = query.Where(task.RemoteNameEQ(remoteName))
	}

	// 使用 WithJobs 配置子查询来获取每个 task 的最新 job
	tasks, err := query.WithJobs(withLatestJobPredicate).All(ctx)

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
