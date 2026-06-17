package domain

import "errors"

// Errores globales de la capa de dominio.
var (
	ErrNotFound     = errors.New("recurso no encontrado")
	ErrUnauthorized = errors.New("no autorizado")

	// ErrEmailTaken, ErrUsernameTaken y ErrDocumentTaken permiten que la capa API distinga una
	// violación de unicidad sin inspeccionar el texto crudo de PostgreSQL (evita filtrar detalles
	// internos). Se usan tanto en la pre-validación de RequestOTP como en el registro definitivo.
	ErrEmailTaken    = errors.New("el correo electrónico ingresado ya se encuentra registrado")
	ErrUsernameTaken = errors.New("el nombre de usuario ingresado ya se encuentra registrado")
	ErrDocumentTaken = errors.New("la cédula ingresada ya se encuentra registrada")
)
