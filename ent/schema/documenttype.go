package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type DocumentType struct {
	ent.Schema
}

func (DocumentType) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "document_type", Schema: "catalogs"},
	}
}

func (DocumentType) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(50).Unique(),
		field.String("name").MaxLen(100),
		field.Time("created_at").Default(time.Now),
	}
}
