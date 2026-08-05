package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Notification struct {
	ent.Schema
}

func (Notification) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "notifications", Schema: "notifications"},
	}
}

func (Notification) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("title").MaxLen(255),
		field.Text("body"),
		field.Bool("is_read").Default(false),
		field.JSON("data", map[string]interface{}{}).Optional(),
		field.Time("created_at").Default(time.Now),
	}
}
