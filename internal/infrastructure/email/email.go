package email

import (
	"context"
	"log/slog"
)

// EmailService define la interfaz para enviar correos electrónicos (ej: OTPs).
type EmailService interface {
	SendOTP(ctx context.Context, email, code string) error
}

type mockEmailService struct{}

// NewMockEmailService crea un servicio de email simulado de grado de desarrollo.
func NewMockEmailService() EmailService {
	return &mockEmailService{}
}

// SendOTP simula el envío imprimiendo el código en los logs estructurados slog.
func (s *mockEmailService) SendOTP(ctx context.Context, email, code string) error {
	slog.Info("----------------------------------------------------------------------")
	slog.Info("MOCK EMAIL SERVICE: SE HA GENERADO Y ENVIADO UN CÓDIGO OTP",
		"email", email,
		"otp_code", code,
	)
	slog.Info("----------------------------------------------------------------------")
	return nil
}
