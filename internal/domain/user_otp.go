package domain

import (
	"context"
	"time"
)

// UserOTP representa la entidad de dominio para un One-Time Password.
type UserOTP struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	OTPCode   string    `json:"otp_code"`
	Action    string    `json:"action"` // 'REGISTER', 'RECOVER', 'DELETE'
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// OTPRepository define los métodos de persistencia para la gestión de OTPs.
type OTPRepository interface {
	CreateOTP(ctx context.Context, otp *UserOTP) error
	ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error
}
