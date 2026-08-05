package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Banner struct {
	ent.Schema
}

func (Banner) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "banners", Schema: "market"},
	}
}

func (Banner) Fields() []ent.Field {
	return []ent.Field{
		field.String("image_url").MaxLen(255).Unique(),
		field.String("link").MaxLen(255),
		field.Bool("is_active").Default(true),
		field.Int("display_order").Default(0),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
