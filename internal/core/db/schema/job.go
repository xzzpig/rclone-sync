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

// Job holds the schema definition for the Job entity.
type Job struct {
	ent.Schema
}

// Fields of the Job.
func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.UUID("task_id", uuid.UUID{}),
		field.Enum("status").
			GoType(model.JobStatus("")).
			Default(string(model.JobStatusPending)),
		field.Enum("trigger").
			GoType(model.JobTrigger("")),
		field.Time("start_time").
			Default(time.Now),
		field.Time("end_time").
			Optional(),
		field.Int("files_transferred").
			Default(0),
		field.Int64("bytes_transferred").
			Default(0),
		field.Int("files_deleted").
			Default(0),
		field.Int("error_count").
			Default(0),
		field.Text("errors").
			Optional(),
	}
}

// Indexes of the Job.
func (Job) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("task_id", "start_time"),
		index.Fields("status"),
	}
}

// Edges of the Job.
func (Job) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).
			Ref("jobs").
			Unique().
			Required().
			Field("task_id"),
		edge.To("logs", JobLog.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
