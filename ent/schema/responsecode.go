package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type ResponseCode struct {
	ent.Schema
}

func (ResponseCode) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "response_codes", Schema: "catalogs"},
	}
}

func (ResponseCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").MaxLen(50).Unique(), // Used as primary key actually, but ent auto adds 'id'
		field.Int("http_status"),
		field.Text("default_message"),
		field.Text("description").Optional(),
	}
}
