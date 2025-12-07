package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
		field.String("remote_name").
			NotEmpty(),
		field.String("remote_path").
			NotEmpty(),
		field.Enum("direction").
			Values("upload", "download", "bidirectional").
			Default("bidirectional"),
		field.String("schedule").
			Optional(),
		field.Bool("realtime").
			Default(false),
		field.JSON("options", map[string]interface{}{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Task.
func (Task) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("jobs", Job.Type),
	}
}
