package middleware

import "net/http"

// CORS habilita el Cross-Origin Resource Sharing (CORS) de manera permisiva
// para facilitar el desarrollo, Swagger UI, Flutter Web y depuraciones locales.
func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Orígenes permitidos: Todos (*)
			w.Header().Set("Access-Control-Allow-Origin", "*")
			
			// Métodos permitidos: GET, POST, PUT, DELETE, OPTIONS
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			
			// Cabeceras permitidas: Accept, Authorization, Content-Type, X-CSRF-Token
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			
			// Cabeceras expuestas: Link
			w.Header().Set("Access-Control-Expose-Headers", "Link")

			// Si es una petición Preflight (OPTIONS), retornar 200 OK de inmediato
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Continuar con el siguiente handler en la cadena
			next.ServeHTTP(w, r)
		})
	}
}
