package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Configuration struct {
	ent.Schema
}

func (Configuration) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "configurations", Schema: "telemetry"},
	}
}

func (Configuration) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").MaxLen(100).Unique(),
		field.JSON("payload", map[string]interface{}{}),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
