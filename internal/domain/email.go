package domain

import "context"

// EmailService define el contrato para el envío de correos electrónicos (ej. OTPs).
type EmailService interface {
	SendOTP(ctx context.Context, email, code string) error
}
