package api

import (
	_ "embed"
	"net/http"
)

// privacyPolicyHTML contiene la página de Política de Privacidad embebida en el binario.
// Se incrusta en tiempo de compilación, por lo que no depende de archivos externos en el
// contenedor de despliegue (Railway).
//
//go:embed assets/privacy.html
var privacyPolicyHTML []byte

// LegalHandler sirve las páginas legales públicas (Política de Privacidad, etc.).
// Estas rutas son públicas y sin autenticación: deben ser accesibles por terceros como
// la consola de Google Play Store, que exige una URL pública a la política de privacidad.
type LegalHandler struct{}

// NewLegalHandler crea una nueva instancia del controlador de páginas legales.
func NewLegalHandler() *LegalHandler {
	return &LegalHandler{}
}

// HandlePrivacyPolicy entrega la Política de Privacidad como una página HTML pública.
// @Summary      Política de Privacidad
// @Description  Página HTML pública con la política de privacidad de Sakoo (requerida por Play Store).
// @Tags         Legal
// @Produce      html
// @Success      200  {string}  string  "Página HTML de la política de privacidad"
// @Router       /privacy [get]
func (h *LegalHandler) HandlePrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Cacheable durante 1 hora: el contenido cambia con muy poca frecuencia.
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(privacyPolicyHTML)
}
