package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type Message struct {
	ent.Schema
}

func (Message) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "messages", Schema: "notifications"},
	}
}

func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("sender_id").Optional(),
		field.Int64("receiver_id").Optional(),
		field.Text("content"),
		field.Time("read_at").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}
