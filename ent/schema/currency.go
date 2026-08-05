package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Currency struct {
	ent.Schema
}

func (Currency) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "currency", Schema: "catalogs"},
	}
}

func (Currency) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(10).Unique(), // Expanded in 007
		field.String("name").MaxLen(100),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
