package middleware

import (
	"net/http"
	"strings"
)

// allowedOrigins define la lista de orígenes explícitamente permitidos.
var allowedOrigins = []string{
	"http://localhost",
	"http://localhost:3000",
	"http://localhost:5000",
	"http://localhost:8080",
	"http://localhost:8081",
	"http://localhost:4200",
	"http://localhost:54321",
}

// isOriginAllowed comprueba si el origen está en la lista blanca.
func isOriginAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	if strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:") {
		return true
	}

	if strings.HasSuffix(origin, ".railway.app") || 
	   strings.HasSuffix(origin, ".up.railway.app") {
		return true
	}

	return false
}

// CORS habilita el Cross-Origin Resource Sharing (CORS) de manera robusta y definitiva.
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			originAllowed := isOriginAllowed(origin)

			if !originAllowed {
				if r.Method == http.MethodOptions {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")

			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else {
				w.Header().Set("Access-Control-Allow-Headers",
					"Accept, Authorization, Content-Type, X-CSRF-Token, Origin, "+
						"X-Requested-With, Content-Length, Accept-Encoding, Cache-Control")
			}

			w.Header().Set("Access-Control-Expose-Headers", "Link, Content-Length, X-Response-Code")
			w.Header().Set("Access-Control-Max-Age", "7200")
			w.Header().Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
