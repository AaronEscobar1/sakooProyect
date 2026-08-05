package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type PaymentNotification struct {
	ent.Schema
}

func (PaymentNotification) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_notifications", Schema: "finance"},
	}
}

func (PaymentNotification) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("payment_commitment_id").Optional(),
		field.Float("amount_paid"),
		field.String("transaction_reference").MaxLen(100),
		field.Time("notification_date").Default(time.Now),
	}
}
