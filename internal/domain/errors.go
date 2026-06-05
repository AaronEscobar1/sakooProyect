package domain

import "errors"

// Errores globales de la capa de dominio.
var (
	ErrNotFound     = errors.New("recurso no encontrado")
	ErrUnauthorized = errors.New("no autorizado")
)
