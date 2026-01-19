package query

import (
	"context"
	"errors"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/taskhook"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
)

// HookQuery provides operations for managing task hooks.
type HookQuery struct {
	client *ent.Client
}

// NewHookQuery creates a new HookQuery instance.
func NewHookQuery(client *ent.Client) *HookQuery {
	return &HookQuery{client: client}
}

// GetHook retrieves a hook by ID.
func (h *HookQuery) GetHook(ctx context.Context, id uuid.UUID) (*ent.TaskHook, error) {
	hook, err := h.client.TaskHook.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hook, nil
}

// ListHooks retrieves hooks based on task, connection, or event.
func (h *HookQuery) ListHooks(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, event *model.HookEvent) ([]*ent.TaskHook, error) {
	query := h.client.TaskHook.Query()

	if taskID != nil {
		query = query.Where(taskhook.TaskIDEQ(*taskID))
	}
	if connectionID != nil {
		query = query.Where(taskhook.ConnectionIDEQ(*connectionID))
	}
	if event != nil {
		query = query.Where(taskhook.EventEQ(*event))
	}

	query = query.Order(taskhook.ByPriority())

	hooks, err := query.All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hooks, nil
}

// CreateHook creates a new hook.
func (h *HookQuery) CreateHook(ctx context.Context, taskID *uuid.UUID, connectionID *uuid.UUID, input model.HookInput) (*ent.TaskHook, error) {
	builder := h.client.TaskHook.Create().
		SetEnabled(input.Enabled == nil || *input.Enabled).
		SetEvent(input.Event).
		SetType(input.Type).
		SetOnError(model.HookOnErrorIgnore).
		SetConfig(model.HookConfigInputToModel(input.Config))

	if input.OnError != nil {
		builder = builder.SetOnError(*input.OnError)
	}
	if input.Priority != nil {
		builder = builder.SetPriority(*input.Priority)
	}
	if taskID != nil {
		builder = builder.SetTaskID(*taskID)
	}
	if connectionID != nil {
		builder = builder.SetConnectionID(*connectionID)
	}

	hook, err := builder.Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, errors.Join(errs.ErrAlreadyExists, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hook, nil
}

// UpdateHook updates an existing hook.
func (h *HookQuery) UpdateHook(ctx context.Context, id uuid.UUID, input model.UpdateHookInput) (*ent.TaskHook, error) {
	builder := h.client.TaskHook.UpdateOneID(id)

	if input.Enabled != nil {
		builder = builder.SetEnabled(*input.Enabled)
	}
	if input.Priority != nil {
		builder = builder.SetPriority(*input.Priority)
	}
	if input.Event != nil {
		builder = builder.SetEvent(*input.Event)
	}
	if input.Type != nil {
		builder = builder.SetType(*input.Type)
	}
	if input.OnError != nil {
		builder = builder.SetOnError(*input.OnError)
	}
	if input.Config != nil {
		builder = builder.SetConfig(model.HookConfigInputToModel(input.Config))
	}

	hook, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hook, nil
}

// DeleteHook deletes a hook by ID.
func (h *HookQuery) DeleteHook(ctx context.Context, id uuid.UUID) (*ent.TaskHook, error) {
	hook, err := h.client.TaskHook.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(errs.ErrNotFound, err)
		}
		return nil, errors.Join(errs.ErrSystem, err)
	}

	err = h.client.TaskHook.DeleteOneID(id).Exec(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hook, nil
}

// GetHooksForEvent retrieves enabled hooks for a specific task and its connection, ordered by priority and type.
func (h *HookQuery) GetHooksForEvent(ctx context.Context, taskID uuid.UUID, connectionID uuid.UUID, event model.HookEvent) ([]*ent.TaskHook, error) {
	query := h.client.TaskHook.Query().
		Where(
			taskhook.Enabled(true),
			taskhook.EventEQ(event),
			taskhook.Or(
				taskhook.TaskIDEQ(taskID),
				taskhook.ConnectionIDEQ(connectionID),
			),
		).
		Order(
			// Order by priority ASC (NULLS LAST), Global hooks first, then creation time
			func(s *sql.Selector) {
				s.OrderBy(
					s.C(taskhook.FieldPriority)+" IS NULL",
					s.C(taskhook.FieldPriority),
					s.C(taskhook.FieldConnectionID)+" IS NULL",
					s.C(taskhook.FieldCreatedAt),
				)
			},
		)

	hooks, err := query.All(ctx)
	if err != nil {
		return nil, errors.Join(errs.ErrSystem, err)
	}
	return hooks, nil
}

var _ ports.HookQuery = (*HookQuery)(nil)
