package middleware

import (
	"net/http"
	"strings"
)

// allowedOrigins define la lista de orígenes explícitamente permitidos.
// En producción esta lista puede cargarse desde variables de entorno.
// Para desarrollo local se permiten todos los puertos de localhost y el swagger host.
var allowedOrigins = []string{
	// Desarrollo local — Swagger UI y Flutter Web
	"http://localhost",
	"http://localhost:3000",
	"http://localhost:5000",
	"http://localhost:8080",
	"http://localhost:8081",
	"http://localhost:4200",
	// Flutter Web típico
	"http://localhost:54321",
}

// isOriginAllowed comprueba si el origen está en la lista blanca.
// Si la lista está vacía, permite cualquier origen (solo para desarrollo).
func isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	// Permitir cualquier localhost aunque no esté explícitamente en la lista
	// (cubre puertos dinámicos de Flutter en desarrollo)
	if strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:") {
		return true
	}
	return false
}

// CORS habilita el Cross-Origin Resource Sharing (CORS) de manera robusta y definitiva.
//
// Diseñado para funcionar con:
//   - Chrome + Swagger UI (usa credenciales Bearer, hace preflights OPTIONS)
//   - Flutter Web (mismo dominio base, distintos puertos dinámicos)
//   - Flutter Mobile / App Nativa (NO envía Origin; no necesita CORS, se permite sin restricción)
//   - Curl / Postman / Bruno (sin Origin; pasa directo sin restricciones)
//
// Reglas críticas implementadas:
//  1. Si Origin está presente y es permitido → se refleja el Origin exacto + Credentials: true
//  2. Si Origin está presente y NO es permitido → se bloquea (no se agregan headers CORS)
//  3. Si Origin está ausente (Flutter mobile, curl) → se permite sin headers CORS (no los necesita)
//  4. Preflight OPTIONS siempre se responde con 204 No Content una vez que pasa la validación de origen
//  5. Max-Age de 7200s para cachear preflights y evitar re-validaciones continuas en Chrome
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// ── Caso 1: petición sin Origin (Flutter mobile, curl, Postman, cron) ──
			// No necesita CORS en absoluto → pasar directamente al handler
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// ── Caso 2: Origin presente pero NO permitido → bloquear preflight, dejar pasar el resto ──
			// Chrome bloqueará la respuesta de todas formas, pero no rompemos la cadena del servidor.
			originAllowed := isOriginAllowed(origin)

			if !originAllowed {
				// Si es preflight de un origen no permitido → rechazar explícitamente
				if r.Method == http.MethodOptions {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}
				// Para peticiones normales de origen no permitido: dejar pasar pero sin headers CORS.
				// El navegador bloqueará la respuesta igualmente por política CORS.
				next.ServeHTTP(w, r)
				return
			}

			// ── Caso 3: Origin válido → inyectar todos los headers CORS correctos ──

			// CRÍTICO: reflejar el Origin exacto (nunca "*") cuando se usa Credentials.
			// Usar "*" + Credentials: true es inválido por spec CORS y Chrome lo rechaza.
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// Permitir envío de credenciales (token Bearer en Authorization).
			// Necesario para Swagger UI y Flutter Web con sesión autenticada.
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Métodos HTTP que el cliente puede utilizar en peticiones reales.
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")

			// Cabeceras que el cliente puede incluir en sus peticiones.
			// Se refleja dinámicamente lo que el browser solicita en el preflight,
			// con un fallback a la lista estándar de la API.
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				// Reflejar exactamente lo que el browser pide → máxima compatibilidad
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers",
					"Accept, Authorization, Content-Type, X-CSRF-Token, Origin, "+
						"X-Requested-With, Content-Length, Accept-Encoding, Cache-Control")
			}

			// Cabeceras que el código JavaScript en el browser puede leer de la respuesta.
			w.Header().Set("Access-Control-Expose-Headers", "Link, Content-Length, X-Response-Code")

			// Cachear el resultado del preflight por 2 horas para evitar que Chrome
			// repita el OPTIONS en cada petición → mejora rendimiento y estabilidad.
			w.Header().Set("Access-Control-Max-Age", "7200")

			// Vary: Origin es obligatorio para que proxies y CDN no sirvan
			// una respuesta CORS cacheada a clientes con distinto Origin.
			w.Header().Add("Vary", "Origin")

			// ── Responder el Preflight (OPTIONS) inmediatamente ──
			// No debe pasar por el router ni por AuthMiddleware.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// ── Petición real: continuar con el pipeline de handlers ──
			next.ServeHTTP(w, r)
		})
	}
}
