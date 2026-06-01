package usecase

import (
	"testing"
)

func TestContainsProfanity(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"Este comentario es normal y constructivo", false},
		{"Este comentario es una mierda total", true},
		{"¡Eres un malparido y un marico!", true},
		{"Qué buena comunicación tenemos", false}, // "marica" no debe coincidir dentro de "comunicación"
		{"El precio del dólar es una verga", true},
		{"¡Puta madre!", true},
		{"hijo de puta", true},
		{"Ese cabrón no sabe de economía", true},
	}

	for _, test := range tests {
		result := containsProfanity(test.input)
		if result != test.expected {
			t.Errorf("Para entrada %q: esperado %v, obtenido %v", test.input, test.expected, result)
		}
	}
}
