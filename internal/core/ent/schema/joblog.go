package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// JobLog holds the schema definition for the JobLog entity.
type JobLog struct {
	ent.Schema
}

// Fields of the JobLog.
func (JobLog) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("level").
			Values("info", "warning", "error"),
		field.Time("time").
			Default(time.Now),
		field.String("path").
			Optional(),
		field.String("message"),
	}
}

// Edges of the JobLog.
func (JobLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("job", Job.Type).
			Ref("logs").
			Unique().
			Required(),
	}
}
