package schema

import (
	"time"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Task holds the schema definition for the Task entity.
type Task struct {
	ent.Schema
}

// Fields of the Task.
func (Task) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("name").
			NotEmpty(),
		field.String("source_path").
			NotEmpty(),
		field.UUID("connection_id", uuid.UUID{}).
			Optional(),
		field.String("remote_path").
			NotEmpty(),
		field.Enum("direction").
			GoType(model.SyncDirection("")).
			Default(string(model.SyncDirectionBidirectional)),
		field.String("schedule").
			Optional(),
		field.Bool("realtime").
			Default(false),
		field.JSON("options", &model.TaskSyncOptions{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Task.
func (Task) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("connection_id"),
		index.Fields("created_at"),
	}
}

// Edges of the Task.
func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("jobs", Job.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("connection", Connection.Type).
			Ref("tasks").
			Unique().
			Field("connection_id"),
	}
}
