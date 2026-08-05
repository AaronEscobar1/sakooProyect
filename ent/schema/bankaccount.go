package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type BankAccount struct {
	ent.Schema
}

func (BankAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "bank_accounts", Schema: "finance"},
	}
}

func (BankAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("bank_name").MaxLen(100),
		field.String("account_number").MaxLen(100),
		field.String("account_type").MaxLen(50),
		field.String("holder_name").MaxLen(100),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
