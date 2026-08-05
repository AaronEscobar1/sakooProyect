package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type UserSession struct {
	ent.Schema
}

func (UserSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_sessions", Schema: "security"},
	}
}

func (UserSession) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("token").MaxLen(255).Unique(),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now),
	}
}
