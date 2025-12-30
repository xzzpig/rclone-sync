// Package schema defines the database schema for the application entities.
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Connection holds the schema definition for the Connection entity.
type Connection struct {
	ent.Schema
}

// Fields of the Connection.
func (Connection) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("name").
			NotEmpty().
			Unique().
			Comment("Remote name, must be unique across the system"),
		field.String("type").
			NotEmpty().
			Comment("Provider type, e.g., onedrive, s3, drive, local"),
		field.Bytes("encrypted_config").
			Comment("AES-GCM encrypted configuration JSON"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Connection.
func (Connection) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
		index.Fields("type"),
		index.Fields("created_at"),
	}
}

// Edges of the Connection.
func (Connection) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tasks", Task.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
