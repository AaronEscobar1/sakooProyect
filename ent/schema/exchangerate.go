package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type ExchangeRate struct {
	ent.Schema
}

func (ExchangeRate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "exchange_rates", Schema: "market"},
	}
}

func (ExchangeRate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("currency_id"),
		field.Float("rate_from"),
		field.Float("rate_to"),
		field.Float("rate_average"),
		field.Time("value_date"),
		field.Time("notified_at").Optional(), // From 004 migration
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
