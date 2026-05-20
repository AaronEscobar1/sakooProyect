package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Configuration representa los parámetros y configuraciones clave del sistema.
type Configuration struct {
	ID        int64
	Key       string          // Clave única identificadora de la configuración
	Payload   json.RawMessage // Datos estructurados en JSON de configuración libre
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConfigurationRepository define los métodos de persistencia para configuraciones globales.
type ConfigurationRepository interface {
	Set(ctx context.Context, config *Configuration) error
	GetByKey(ctx context.Context, key string) (*Configuration, error)
}
