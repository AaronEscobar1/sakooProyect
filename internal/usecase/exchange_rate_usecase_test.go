package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/shopspring/decimal"
)

// mockExchangeRateRepository implements domain.ExchangeRateRepository
type mockExchangeRateRepository struct {
	domain.ExchangeRateRepository // Embed to avoid implementing all methods
	updateRateApprovalFunc       func(ctx context.Context, rateID int64, rateFrom, rateTo, rateAverage decimal.Decimal, source string) error
	getLast7DaysRatesFunc        func(ctx context.Context) ([]domain.ExchangeRate, error)
}

func (m *mockExchangeRateRepository) UpdateRateApproval(
	ctx context.Context,
	rateID int64,
	rateFrom, rateTo, rateAverage decimal.Decimal,
	source string,
) error {
	if m.updateRateApprovalFunc != nil {
		return m.updateRateApprovalFunc(ctx, rateID, rateFrom, rateTo, rateAverage, source)
	}
	return nil
}

func (m *mockExchangeRateRepository) GetLast7DaysRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	if m.getLast7DaysRatesFunc != nil {
		return m.getLast7DaysRatesFunc(ctx)
	}
	return nil, nil
}

func TestApproveRate(t *testing.T) {
	validRateFrom := decimal.NewFromFloat(50.5)
	validRateTo := decimal.NewFromFloat(51.2)
	validRateAverage := decimal.NewFromFloat(50.85)

	tests := []struct {
		name          string
		req           domain.ApproveExchangeRateRequest
		adminUserID   int64
		setupRepo     func(repo *mockExchangeRateRepository)
		expectedErr   string
		expectSuccess bool
	}{
		{
			name: "Success approval",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    validRateFrom,
				RateTo:      validRateTo,
				RateAverage: validRateAverage,
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo: func(repo *mockExchangeRateRepository) {
				repo.updateRateApprovalFunc = func(ctx context.Context, rateID int64, rateFrom, rateTo, rateAverage decimal.Decimal, source string) error {
					if rateID != 100 || !rateFrom.Equal(validRateFrom) || !rateTo.Equal(validRateTo) || !rateAverage.Equal(validRateAverage) || source != "MANUAL" {
						return errors.New("unexpected mock inputs")
					}
					return nil
				}
			},
			expectSuccess: true,
		},
		{
			name: "Failure - RateID <= 0",
			req: domain.ApproveExchangeRateRequest{
				RateID:      0,
				RateFrom:    validRateFrom,
				RateTo:      validRateTo,
				RateAverage: validRateAverage,
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo:   func(repo *mockExchangeRateRepository) {},
			expectedErr: "el ID de la tasa de cambio es requerido y debe ser un número positivo",
		},
		{
			name: "Failure - RateFrom negative",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    decimal.NewFromFloat(-1.0),
				RateTo:      validRateTo,
				RateAverage: validRateAverage,
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo:   func(repo *mockExchangeRateRepository) {},
			expectedErr: "el campo rate_from debe ser un valor positivo mayor a cero",
		},
		{
			name: "Failure - RateTo zero",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    validRateFrom,
				RateTo:      decimal.NewFromFloat(0.0),
				RateAverage: validRateAverage,
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo:   func(repo *mockExchangeRateRepository) {},
			expectedErr: "el campo rate_to debe ser un valor positivo mayor a cero",
		},
		{
			name: "Failure - RateAverage zero",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    validRateFrom,
				RateTo:      validRateTo,
				RateAverage: decimal.NewFromFloat(0),
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo:   func(repo *mockExchangeRateRepository) {},
			expectedErr: "el campo rate_average debe ser un valor positivo mayor a cero",
		},
		{
			name: "Failure - Source empty",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    validRateFrom,
				RateTo:      validRateTo,
				RateAverage: validRateAverage,
				Source:      "",
			},
			adminUserID: 99,
			setupRepo:   func(repo *mockExchangeRateRepository) {},
			expectedErr: "el campo source es requerido (ej: 'MANUAL', 'SCRAPING')",
		},
		{
			name: "Failure - Repo error propagates",
			req: domain.ApproveExchangeRateRequest{
				RateID:      100,
				RateFrom:    validRateFrom,
				RateTo:      validRateTo,
				RateAverage: validRateAverage,
				Source:      "MANUAL",
			},
			adminUserID: 99,
			setupRepo: func(repo *mockExchangeRateRepository) {
				repo.updateRateApprovalFunc = func(ctx context.Context, rateID int64, rateFrom, rateTo, rateAverage decimal.Decimal, source string) error {
					return errors.New("db write failed")
				}
			},
			expectedErr: "db write failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockExchangeRateRepository{}
			tt.setupRepo(repo)

			uc := usecase.NewExchangeRateUseCase(repo)
			err := uc.ApproveRate(context.Background(), tt.req, tt.adminUserID)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
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

func TestGetLast7DaysRates(t *testing.T) {
	t.Run("Success fetching rates", func(t *testing.T) {
		repo := &mockExchangeRateRepository{
			getLast7DaysRatesFunc: func(ctx context.Context) ([]domain.ExchangeRate, error) {
				return []domain.ExchangeRate{
					{ID: 1, CurrencyCode: "USD", Status: "APPROVED"},
					{ID: 2, CurrencyCode: "EUR", Status: "REGISTERED"},
				}, nil
			},
		}

		uc := usecase.NewExchangeRateUseCase(repo)
		rates, err := uc.GetLast7DaysRates(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(rates) != 2 {
			t.Errorf("expected 2 rates, got %d", len(rates))
		}

		if rates[0].ID != 1 || rates[1].ID != 2 {
			t.Errorf("unexpected rates content")
		}
	})

	t.Run("Failure propagates error", func(t *testing.T) {
		expectedErr := errors.New("db disconnect")
		repo := &mockExchangeRateRepository{
			getLast7DaysRatesFunc: func(ctx context.Context) ([]domain.ExchangeRate, error) {
				return nil, expectedErr
			},
		}

		uc := usecase.NewExchangeRateUseCase(repo)
		_, err := uc.GetLast7DaysRates(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
