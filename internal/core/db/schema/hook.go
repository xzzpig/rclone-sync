package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"
)

type TaskHook struct {
	ent.Schema
}

func (TaskHook) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.Bool("enabled").
			Default(true),
		field.Int("priority").
			Optional().
			Nillable(),
		field.Enum("event").
			GoType(model.HookEvent("")),
		field.Enum("type").
			GoType(model.HookType("")),
		field.Enum("on_error").
			GoType(model.HookOnError("")).
			Default(string(model.HookOnErrorIgnore)),
		field.JSON("config", &model.HookConfig{}),
		field.UUID("task_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.UUID("connection_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (TaskHook) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("connection_id"),
		index.Fields("task_id", "event", "priority", "created_at"),
		index.Fields("connection_id", "event", "priority", "created_at"),
		index.Fields("created_at"),
	}
}

func (TaskHook) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).
			Ref("hooks").
			Unique().
			Field("task_id"),
		edge.From("connection", Connection.Type).
			Ref("hooks").
			Unique().
			Field("connection_id"),
	}
}
