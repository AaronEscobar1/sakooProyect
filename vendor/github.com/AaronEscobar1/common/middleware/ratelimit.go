package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AaronEscobar1/common/response"
)

// rateLimiter es un limitador de tasa en memoria con ventana deslizante (sliding-log).
// Mantiene, por clave (IP del cliente), las marcas de tiempo de las últimas peticiones
// dentro de la ventana. Es de bajo costo y no requiere dependencias externas (Redis, etc.),
// adecuado para un único nodo. Para despliegues multi-instancia conviene migrar a un store compartido.
type rateLimiter struct {
	mu       sync.Mutex
	hits     map[string][]time.Time
	max      int
	window   time.Duration
	lastSeen map[string]time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		hits:     make(map[string][]time.Time),
		max:      max,
		window:   window,
		lastSeen: make(map[string]time.Time),
	}
	go rl.cleanupLoop()
	return rl
}

// allow registra un intento para la clave y devuelve false si se superó el límite en la ventana.
func (rl *rateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Conservar solo las marcas dentro de la ventana vigente
	recent := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	rl.lastSeen[key] = now

	if len(recent) >= rl.max {
		rl.hits[key] = recent
		return false
	}

	rl.hits[key] = append(recent, now)
	return true
}

// cleanupLoop purga periódicamente las claves inactivas para evitar crecimiento ilimitado de memoria.
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rl.window)
		rl.mu.Lock()
		for key, seen := range rl.lastSeen {
			if seen.Before(cutoff) {
				delete(rl.hits, key)
				delete(rl.lastSeen, key)
			}
		}
		rl.mu.Unlock()
	}
}

// clientIP extrae la IP real del cliente respetando los headers de proxy (Cloudflare/Railway).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// RateLimit devuelve un middleware que limita las peticiones a `max` por `window` y por IP de cliente.
// Cada invocación crea un store independiente (un "cubo" por endpoint protegido).
func RateLimit(max int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(max, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Las solicitudes de preflight CORS no deben consumir cuota.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			if !rl.allow(clientIP(r)) {
				response.Error(w, r.Context(), http.StatusTooManyRequests, "TOO_MANY_REQUESTS",
					"Demasiados intentos. Por favor, espera unos minutos antes de volver a intentarlo.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
