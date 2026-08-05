package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type UserPasswordHistory struct {
	ent.Schema
}

func (UserPasswordHistory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_passwords_history", Schema: "security"},
	}
}

func (UserPasswordHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("password_hash").MaxLen(255),
		field.Time("created_at").Default(time.Now),
	}
}
