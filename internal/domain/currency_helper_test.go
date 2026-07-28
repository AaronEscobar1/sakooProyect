package domain_test

import (
	"testing"

	"github.com/aaron/sakoo-backend/internal/domain"
)

func TestCleanCurrencyCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "código sin espacios",
			input:    "USDT",
			expected: "USDT",
		},
		{
			name:     "código concatenado USDT - TETHER",
			input:    "USDT - TETHER",
			expected: "USDT",
		},
		{
			name:     "código concatenado USD - Dólar Estadounidense",
			input:    "USD - Dólar Estadounidense",
			expected: "USD",
		},
		{
			name:     "espacios al inicio y al final",
			input:    "  EUR - Euro  ",
			expected: "EUR",
		},
		{
			name:     "cadena vacía",
			input:    "",
			expected: "",
		},
		{
			name:     "sólo espacios",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.CleanCurrencyCode(tt.input)
			if result != tt.expected {
				t.Errorf("CleanCurrencyCode(%q) = %q, se esperaba %q", tt.input, result, tt.expected)
			}
		})
	}
}
