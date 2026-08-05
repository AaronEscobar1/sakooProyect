package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Bank struct {
	ent.Schema
}

func (Bank) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "banks", Schema: "catalogs"},
	}
}

func (Bank) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(20).Unique(),
		field.String("name").MaxLen(100),
		field.Bool("show").Default(true),
		field.Time("created_at").Default(time.Now),
	}
}
