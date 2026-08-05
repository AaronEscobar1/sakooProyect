package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"time"
)

type ApiLog struct {
	ent.Schema
}

func (ApiLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "api_logs", Schema: "telemetry"},
	}
}

func (ApiLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("track_code").MaxLen(50),
		field.Int64("user_id").Optional(),
		field.String("method").MaxLen(10),
		field.String("path").MaxLen(255),
		field.Int("http_status"),
		field.String("response_code").MaxLen(50).Optional(),
		field.Int64("latency_ms"),
		field.Time("created_at").Default(time.Now),
	}
}
