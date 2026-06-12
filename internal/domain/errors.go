package domain

import "errors"

// Errores globales de la capa de dominio.
var (
	ErrNotFound     = errors.New("recurso no encontrado")
	ErrUnauthorized = errors.New("no autorizado")

	// ErrEmailTaken y ErrUsernameTaken permiten que la capa API distinga una violación de unicidad
	// sin tener que inspeccionar el texto crudo del error de PostgreSQL (evita filtrar detalles internos).
	ErrEmailTaken    = errors.New("el correo electrónico ingresado ya se encuentra registrado")
	ErrUsernameTaken = errors.New("el nombre de usuario ingresado ya se encuentra registrado")
)
