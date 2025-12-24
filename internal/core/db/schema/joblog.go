package schema

import (
	"time"

	"github.com/xzzpig/rclone-sync/internal/api/graphql/model"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// JobLog holds the schema definition for the JobLog entity.
type JobLog struct {
	ent.Schema
}

// Fields of the JobLog.
func (JobLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("job_id", uuid.UUID{}),
		field.Enum("level").
			GoType(model.LogLevel("")),
		field.Time("time").
			Default(time.Now),
		field.String("path").
			Optional(),
		field.Enum("what").
			GoType(model.LogAction("")).
			Default(string(model.LogActionUnknown)),
		field.Int64("size").
			Optional(),
	}
}

// Edges of the JobLog.
func (JobLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("job", Job.Type).
			Ref("logs").
			Unique().
			Required().
			Field("job_id"),
	}
}
