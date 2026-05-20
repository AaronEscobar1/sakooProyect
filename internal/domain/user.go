package domain

import (
	"context"
	"time"
)

// UserType representa el tipo de usuario (ej: ADMIN, CUSTOMER).
type UserType struct {
	ID        int64
	Code      string
	Name      string
	CreatedAt time.Time
}

// DocumentType representa el tipo de documento de identidad (ej: DNI, PASSPORT).
type DocumentType struct {
	ID        int64
	Code      string
	Name      string
	CreatedAt time.Time
}

// User representa a un usuario registrado en el sistema.
type User struct {
	ID             int64
	Email          string
	Username       string
	FirstName      string
	LastName       string
	MiddleName     *string
	SecondLastName *string
	AvatarIndex    int
	UserTypeID     int64
	DocumentTypeID *int64 // Puntero para admitir valores nulos
	DocumentNumber *string // Puntero para admitir valores nulos
	PasswordHash   string  // Contraseña encriptada con bcrypt
	RegistrationIP *string
	Country        *string
	DeletedAt      *time.Time // Borrado lógico
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DTOs para los flujos de autenticación
type RegisterRequest struct {
	Email          string  `json:"email"`
	Username       string  `json:"username"`
	Password       string  `json:"password"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	MiddleName     *string `json:"middle_name,omitempty"`
	SecondLastName *string `json:"second_last_name,omitempty"`
	UserTypeID     int64   `json:"user_type_id"`
	DocumentTypeID *int64  `json:"document_type_id,omitempty"`
	DocumentNumber *string `json:"document_number,omitempty"`
	OTPCode        string  `json:"otp_code"`
	RegistrationIP string  `json:"-"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

// UserRepository define la interfaz de persistencia para el módulo de usuarios.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	SoftDelete(ctx context.Context, userID int64) error
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	GetPasswordHistory(ctx context.Context, userID int64) ([]string, error)
	AddPasswordHistory(ctx context.Context, userID int64, passwordHash string) error
}

// AuthUseCase define la lógica de negocio para el módulo de autenticación.
type AuthUseCase interface {
	Register(ctx context.Context, req RegisterRequest) error
	Login(ctx context.Context, req LoginRequest) (AuthResponse, error)
	DeleteMyAccount(ctx context.Context, userID int64, email, password string) error
	RequestOTP(ctx context.Context, email, action string) error
	ResetPassword(ctx context.Context, email, newPassword, otpCode string) error
	DeleteAccount(ctx context.Context, userID int64, otpCode string) error
	GetProfile(ctx context.Context, userID int64) (*User, error)
}
