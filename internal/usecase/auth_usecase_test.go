package usecase

import (
	"testing"
)

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
		desc     string
	}{
		{"S1!", false, "Demasiado corta"},
		{"nouppercase1!", false, "Sin mayúscula"},
		{"NOLOWERCASE1!", false, "Sin minúscula"},
		{"NoNumber!", false, "Sin número"},
		{"NoSpecialChar1", false, "Sin carácter especial"},
		{"ValidPassword1!", true, "Contraseña válida estándar"},
		{"Another$Valid9", true, "Contraseña válida con $"},
		{"P@ssw0rdStrength", true, "Contraseña válida con @"},
	}

	for _, tc := range tests {
		err := validatePasswordStrength(tc.password)
		if tc.valid && err != nil {
			t.Errorf("Test '%s' falló: Esperaba contraseña %q válida, pero obtuve error: %v", tc.desc, tc.password, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("Test '%s' falló: Esperaba contraseña %q inválida, pero no obtuve error", tc.desc, tc.password)
		}
	}
}
