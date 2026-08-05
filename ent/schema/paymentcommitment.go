package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type PaymentCommitment struct {
	ent.Schema
}

func (PaymentCommitment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_commitments", Schema: "finance"},
	}
}

func (PaymentCommitment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").Optional(),
		field.Float("amount"),
		field.Int64("currency_id").Optional(),
		field.Time("due_date"),
		field.String("status").MaxLen(50),
		field.Time("created_at").Default(time.Now),
	}
}
