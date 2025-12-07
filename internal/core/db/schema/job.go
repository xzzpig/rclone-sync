package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
		field.Enum("status").
			Values("pending", "running", "success", "failed", "cancelled").
			Default("pending"),
		field.Enum("trigger").
			Values("manual", "schedule", "realtime"),
		field.Time("start_time").
			Default(time.Now),
		field.Time("end_time").
			Optional(),
		field.Int("files_transferred").
			Default(0),
		field.Int64("bytes_transferred").
			Default(0),
		field.Text("errors").
			Optional(),
	}
}

// Edges of the Job.
func (Job) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).
			Ref("jobs").
			Unique().
			Required(),
		edge.To("logs", JobLog.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
