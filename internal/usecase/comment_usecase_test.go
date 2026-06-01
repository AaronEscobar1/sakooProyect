package usecase

import (
	"testing"
)

func TestCensorText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Este comentario es normal y constructivo", "Este comentario es normal y constructivo"},
		{"Este comentario es una mierda total", "Este comentario es una **** total"},
		{"¡Eres un malparido y un marico!", "¡Eres un **** y un ****!"},
		{"Qué buena comunicación tenemos", "Qué buena comunicación tenemos"}, // "marica" no debe coincidir dentro de "comunicación"
		{"El precio del dólar es una verga", "El precio del dólar es una ****"},
		{"¡Puta madre!", "¡**** madre!"},
		{"hijo de puta", "****"},
		{"Ese cabrón no sabe de economía", "Ese **** no sabe de economía"},
		{"Ese mmg no sabe nada", "Ese **** no sabe nada"},
		{"¡csm todo esto!", "¡**** todo esto!"},
		{"Que mariko eres de verdad", "Que **** eres de verdad"},
		{"El microcosmos de la economía", "El microcosmos de la economía"}, // "csm" no debe coincidir dentro de "microcosmos"
		{"La verdad no quiero saber nada de estos mmgvos", "La verdad no quiero saber nada de estos ****"},
		{"esto es un experimento de las groserías, marico pajuo pajuato mmgvo mmgvos maldito idiota mamaguevo hijo de perra chupa culo marginal maldito marico mamaguebaso teta hijo de puta tetas", "esto es un experimento de las groserías, **** **** **** **** **** **** **** **** **** **** **** **** **** **** **** **** ****"},
		{"Ese becerro y sapo es un tremendo enchufado, tremenda totona", "Ese **** y **** es un tremendo ****, tremenda ****"},
	}

	for _, test := range tests {
		result := censorText(test.input)
		if result != test.expected {
			t.Errorf("Para entrada %q: esperado %q, obtenido %q", test.input, test.expected, result)
		}
	}
}
