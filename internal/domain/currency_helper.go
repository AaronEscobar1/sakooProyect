package domain

import "strings"

// CleanCurrencyCode toma un código de moneda crudo (ej: "USDT - TETHER", "USD", " EUR - Euro ")
// y extrae únicamente el código base cortando la cadena en el primer espacio en blanco.
func CleanCurrencyCode(raw string) string {
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return ""
	}
	if idx := strings.Index(cleaned, " "); idx != -1 {
		return strings.TrimSpace(cleaned[:idx])
	}
	return cleaned
}
