package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users", Schema: "security"},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").MaxLen(255).Unique(),
		field.String("username").MaxLen(100).Unique().Optional(),
		field.String("first_name").MaxLen(100),
		field.String("middle_name").MaxLen(100).Optional(),
		field.String("last_name").MaxLen(100),
		field.String("second_last_name").MaxLen(100).Optional(),
		field.Int("avatar_index").Default(0),
		field.Int64("user_type_id"),
		field.Int64("document_type_id").Optional(),
		field.String("document_number").MaxLen(50).Optional(),
		field.String("password_hash").MaxLen(255),
		field.String("registration_ip").MaxLen(50).Optional(),
		field.String("country").MaxLen(100).Optional(),
		field.Time("deleted_at").Optional(),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
