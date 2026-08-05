package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type UserDeviceToken struct {
	ent.Schema
}

func (UserDeviceToken) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_device_tokens", Schema: "notifications"},
	}
}

func (UserDeviceToken) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.Text("token").Unique(),
		field.String("platform").MaxLen(50),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
