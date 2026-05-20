package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type otpRepository struct {
	db *pgxpool.Pool
}

// NewOTPRepository crea una nueva instancia del repositorio de OTPs.
func NewOTPRepository(db *pgxpool.Pool) domain.OTPRepository {
	return &otpRepository{
		db: db,
	}
}

// CreateOTP guarda un nuevo OTP en la base de datos PostgreSQL.
func (r *otpRepository) CreateOTP(ctx context.Context, otp *domain.UserOTP) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Guardando OTP en base de datos", "email", otp.Email, "action", otp.Action)

	query := `
		INSERT INTO public.user_otps (
			email, 
			otp_code, 
			action, 
			expires_at, 
			used, 
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at;
	`
	err := r.db.QueryRow(dbCtx, query, 
		otp.Email, 
		otp.OTPCode, 
		otp.Action, 
		otp.ExpiresAt, 
		otp.Used,
	).Scan(&otp.ID, &otp.CreatedAt)

	if err != nil {
		slog.Error("Fallo al guardar OTP en PostgreSQL", "error", err, "email", otp.Email)
		return fmt.Errorf("error al guardar OTP en base de datos: %w", err)
	}

	slog.Info("OTP registrado exitosamente", "id", otp.ID, "email", otp.Email, "action", otp.Action)
	return nil
}

// ValidateAndConsumeOTP busca y consume atómicamente el OTP para evitar Race Conditions.
func (r *otpRepository) ValidateAndConsumeOTP(ctx context.Context, email, code, action string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Validando y consumiendo OTP de forma atómica", "email", email, "action", action)

	query := `
		UPDATE public.user_otps
		SET used = true
		WHERE email = $1 AND otp_code = $2 AND action = $3 AND used = false AND expires_at > (now() at time zone 'utc')
		RETURNING id;
	`

	var id int64
	err := r.db.QueryRow(dbCtx, query, email, code, action).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("Intento de validación fallido: OTP inválido, expirado o ya consumido", "email", email, "action", action)
			return errors.New("código OTP inválido, expirado o ya consumido")
		}
		slog.Error("Error al consumir OTP en PostgreSQL", "error", err, "email", email)
		return fmt.Errorf("error al verificar el código OTP: %w", err)
	}

	slog.Info("OTP validado y consumido correctamente", "id", id, "email", email, "action", action)
	return nil
}
