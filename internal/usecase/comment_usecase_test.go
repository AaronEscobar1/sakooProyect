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
		{"Ese mmg no sabe nada", true},
		{"¡csm todo esto!", true},
		{"Que mariko eres de verdad", true},
		{"El microcosmos de la economía", false}, // "csm" no debe coincidir dentro de "microcosmos"
	}

	for _, test := range tests {
		result := containsProfanity(test.input)
		if result != test.expected {
			t.Errorf("Para entrada %q: esperado %v, obtenido %v", test.input, test.expected, result)
		}
	}
}
