package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aaron/sakoo-backend/internal/api/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestAdminOnly(t *testing.T) {
	jwtSecret := "test-secret-key"

	// Helper to generate test JWT tokens
	generateToken := func(userType string, expired bool) string {
		exp := time.Now().Add(1 * time.Hour).Unix()
		if expired {
			exp = time.Now().Add(-1 * time.Hour).Unix()
		}

		claims := jwt.MapClaims{
			"user_id":   float64(123),
			"exp":       exp,
			"iat":       time.Now().Unix(),
		}
		if userType != "" {
			claims["user_type"] = userType
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenStr, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		return tokenStr
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid Authorization Header Format",
			authHeader:     "InvalidTokenHere",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Invalid Authorization Header Prefix",
			authHeader:     "Basic abc123def456",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Expired Token",
			authHeader:     fmt.Sprintf("Bearer %s", generateToken("ADMIN", true)),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Valid Token but Missing user_type Claim",
			authHeader:     fmt.Sprintf("Bearer %s", generateToken("", false)),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Valid Token but CUSTOMER user_type Claim",
			authHeader:     fmt.Sprintf("Bearer %s", generateToken("CUSTOMER", false)),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Valid Token and ADMIN user_type Claim",
			authHeader:     fmt.Sprintf("Bearer %s", generateToken("ADMIN", false)),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create next handler
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Instantiate middleware
			mw := middleware.AdminOnly(jwtSecret)(nextHandler)

			// Create test request
			req := httptest.NewRequest(http.MethodPut, "/api/backoffice/rates/approve", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)

			responseCode := rec.Header().Get("X-Response-Code")

			if tt.expectedStatus == http.StatusForbidden {
				if responseCode != "FORBIDDEN" {
					t.Errorf("expected X-Response-Code header to be 'FORBIDDEN', got '%s'", responseCode)
				}
			} else {
				if responseCode == "FORBIDDEN" {
					t.Errorf("expected access to be allowed, but X-Response-Code was 'FORBIDDEN'")
				}
				if rec.Code != http.StatusOK {
					t.Errorf("expected HTTP status 200 for successful request, got %d", rec.Code)
				}
			}
		})
	}
}
