package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository implements domain.UserRepository for unit testing
type mockUserRepository struct {
	domain.UserRepository // Embed to avoid implementing all methods
	findByEmailFunc      func(ctx context.Context, email string) (*domain.User, error)
	createSessionFunc    func(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	getUserTypeCodeFunc  func(ctx context.Context, userTypeID int64) (string, error)
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailFunc != nil {
		return m.findByEmailFunc(ctx, email)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepository) CreateSession(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, userID, token, expiresAt)
	}
	return nil
}

func (m *mockUserRepository) GetUserTypeCode(ctx context.Context, userTypeID int64) (string, error) {
	if m.getUserTypeCodeFunc != nil {
		return m.getUserTypeCodeFunc(ctx, userTypeID)
	}
	return "", errors.New("not implemented")
}

func (m *mockUserRepository) DeleteUserSessions(ctx context.Context, userID int64) error {
	return nil
}

func (m *mockUserRepository) ExtendSession(ctx context.Context, token string, newExpiresAt time.Time) error {
	return nil
}

// mockOTPRepository implements domain.OTPRepository
type mockOTPRepository struct {
	domain.OTPRepository
}

// mockEmailService implements domain.EmailService
type mockEmailService struct {
	domain.EmailService
}

// mockNotificationRepository implements domain.NotificationRepository
type mockNotificationRepository struct {
	domain.NotificationRepository
}

func TestLoginAdmin(t *testing.T) {
	jwtSecret := "test-secret"
	password := "my-secure-password"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	defaultUser := &domain.User{
		ID:           123,
		Email:        "admin@sakoo.com",
		PasswordHash: string(hash),
		UserTypeID:   1,
	}

	tests := []struct {
		name          string
		req           domain.LoginAdminRequest
		setupRepo     func(repo *mockUserRepository)
		expectedErr   string
		expectSuccess bool
	}{
		{
			name: "Success Admin Login",
			req: domain.LoginAdminRequest{
				Email:         "admin@sakoo.com",
				Password:      password,
				RequiresAdmin: true,
			},
			setupRepo: func(repo *mockUserRepository) {
				repo.findByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
					return defaultUser, nil
				}
				repo.getUserTypeCodeFunc = func(ctx context.Context, id int64) (string, error) {
					return "ADMIN", nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "Failure - Email and Password empty",
			req: domain.LoginAdminRequest{
				Email:         "",
				Password:      "",
				RequiresAdmin: true,
			},
			setupRepo:   func(repo *mockUserRepository) {},
			expectedErr: "El correo electrónico y la contraseña son requeridos",
		},
		{
			name: "Failure - User Not Found (Enumeration Mitigation)",
			req: domain.LoginAdminRequest{
				Email:         "nonexistent@sakoo.com",
				Password:      password,
				RequiresAdmin: true,
			},
			setupRepo: func(repo *mockUserRepository) {
				repo.findByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
					return nil, errors.New("user not found")
				}
			},
			expectedErr: "Credenciales incorrectas",
		},
		{
			name: "Failure - Wrong Password",
			req: domain.LoginAdminRequest{
				Email:         "admin@sakoo.com",
				Password:      "wrong-password",
				RequiresAdmin: true,
			},
			setupRepo: func(repo *mockUserRepository) {
				repo.findByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
					return defaultUser, nil
				}
			},
			expectedErr: "Credenciales incorrectas",
		},
		{
			name: "Failure - User is CUSTOMER (Enumeration Mitigation)",
			req: domain.LoginAdminRequest{
				Email:         "customer@sakoo.com",
				Password:      password,
				RequiresAdmin: true,
			},
			setupRepo: func(repo *mockUserRepository) {
				repo.findByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
					return &domain.User{
						ID:           456,
						Email:        "customer@sakoo.com",
						PasswordHash: string(hash),
						UserTypeID:   2,
					}, nil
				}
				repo.getUserTypeCodeFunc = func(ctx context.Context, id int64) (string, error) {
					return "CUSTOMER", nil
				}
			},
			expectedErr: "Credenciales incorrectas",
		},
		{
			name: "Failure - GetUserTypeCode DB Error (Enumeration Mitigation)",
			req: domain.LoginAdminRequest{
				Email:         "admin@sakoo.com",
				Password:      password,
				RequiresAdmin: true,
			},
			setupRepo: func(repo *mockUserRepository) {
				repo.findByEmailFunc = func(ctx context.Context, email string) (*domain.User, error) {
					return defaultUser, nil
				}
				repo.getUserTypeCodeFunc = func(ctx context.Context, id int64) (string, error) {
					return "", errors.New("database connection lost")
				}
			},
			expectedErr: "Credenciales incorrectas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{}
			tt.setupRepo(repo)

			uc := usecase.NewAuthUseCase(repo, &mockOTPRepository{}, &mockEmailService{}, &mockNotificationRepository{}, jwtSecret)
			res, err := uc.LoginAdmin(context.Background(), tt.req)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				if res.Token == "" {
					t.Fatalf("expected JWT token, got empty string")
				}
			} else {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tt.expectedErr)
				}
				if err.Error() != tt.expectedErr {
					t.Errorf("expected error message '%s', got '%s'", tt.expectedErr, err.Error())
				}
			}
		})
	}
}
