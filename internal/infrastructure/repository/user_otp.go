package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/userotp"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type otpRepository struct {
	client *ent.Client
}

// NewOTPRepository crea una nueva instancia del repositorio de OTPs con Ent.
func NewOTPRepository(client *ent.Client) domain.OTPRepository {
	return &otpRepository{
		client: client,
	}
}

// CreateOTP guarda un nuevo OTP en la base de datos PostgreSQL usando Ent.
func (r *otpRepository) CreateOTP(ctx context.Context, otp *domain.UserOTP) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Guardando OTP en base de datos", "email", otp.Email, "action", otp.Action)

	created, err := r.client.UserOtp.Create().
		SetEmail(otp.Email).
		SetOtpCode(otp.OTPCode).
		SetAction(otp.Action).
		SetExpiresAt(otp.ExpiresAt).
		SetUsed(otp.Used).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al guardar OTP en Ent", "error", err, "email", otp.Email)
		return fmt.Errorf("error al guardar OTP en base de datos")
	}

	otp.ID = int64(created.ID)
	otp.CreatedAt = created.CreatedAt

	slog.Info("OTP registrado exitosamente", "id", otp.ID, "email", otp.Email, "action", otp.Action)
	return nil
}

// ValidateAndConsumeOTP busca y consume atómicamente el OTP para evitar Race Conditions.
func (r *otpRepository) ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Validando y consumiendo OTP de forma atómica", "email", email, "action", action)

	count, err := r.client.UserOtp.Update().
		Where(
			userotp.EmailEQ(email),
			userotp.OtpCodeEQ(code),
			userotp.ActionEQ(action),
			userotp.UsedEQ(false),
			userotp.ExpiresAtGT(time.Now()),
		).
		SetUsed(true).
		Save(dbCtx)

	if err != nil {
		slog.Error("Error al consumir OTP en Ent", "error", err, "email", email)
		return fmt.Errorf("Error al verificar el código OTP")
	}

	if count == 0 {
		slog.Warn("Intento de validación fallido: OTP inválido, expirado o ya consumido", "email", email, "action", action)
		return errors.New("Código OTP inválido, expirado o ya consumido")
	}

	slog.Info("OTP validado y consumido correctamente", "email", email, "action", action)
	return nil
}

// ValidateOTPOnly valida la existencia y vigencia de un OTP sin consumirlo.
func (r *otpRepository) ValidateOTPOnly(ctx context.Context, email, code, action string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Validando OTP sin consumirlo", "email", email, "action", action)

	exists, err := r.client.UserOtp.Query().
		Where(
			userotp.EmailEQ(email),
			userotp.OtpCodeEQ(code),
			userotp.ActionEQ(action),
			userotp.UsedEQ(false),
			userotp.ExpiresAtGT(time.Now()),
		).
		Exist(dbCtx)

	if err != nil {
		slog.Error("Error al validar OTP en Ent", "error", err, "email", email)
		return fmt.Errorf("Error al verificar el código OTP")
	}

	if !exists {
		slog.Warn("Intento de validación de OTP fallido: OTP inválido, expirado o ya consumido", "email", email, "action", action)
		return errors.New("Código OTP inválido, expirado o ya consumido")
	}

	slog.Info("OTP validado correctamente (sin consumir)", "email", email, "action", action)
	return nil
}

func (r *otpRepository) HasRecentOTP(ctx context.Context, email, action string, seconds int) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Verificando si se solicitó un OTP recientemente", "email", email, "action", action, "seconds", seconds)

	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)

	exists, err := r.client.UserOtp.Query().
		Where(
			userotp.EmailEQ(email),
			userotp.ActionEQ(action),
			userotp.CreatedAtGT(cutoff),
		).
		Exist(dbCtx)

	if err != nil {
		slog.Error("Fallo al comprobar OTP reciente en Ent", "error", err, "email", email)
		return false, fmt.Errorf("error al verificar OTP reciente")
	}

	return exists, nil
}
