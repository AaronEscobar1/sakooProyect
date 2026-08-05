package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type UserOtp struct {
	ent.Schema
}

func (UserOtp) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user_otps", Schema: "security"},
	}
}

func (UserOtp) Fields() []ent.Field {
	return []ent.Field{
		field.String("email").MaxLen(255),
		field.String("otp_code").MaxLen(10),
		field.String("action").MaxLen(50),
		field.Time("expires_at"),
		field.Bool("used").Default(false),
		field.Time("created_at").Default(time.Now),
	}
}
