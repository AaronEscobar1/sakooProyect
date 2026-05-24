package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Clave de contexto privada para evitar colisiones con otras bibliotecas
type contextKey struct{}

var userContextKey = contextKey{}

// generateTrackCode genera un identificador corto y único de 16 caracteres alfanuméricos
func generateTrackCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// loggingResponseWriter envuelve http.ResponseWriter para capturar el status code e info de headers
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// TraceAndLogMiddleware genera un código de trazabilidad único por petición y registra logs asíncronos en base de datos.
func TraceAndLogMiddleware(db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			trackCode := generateTrackCode()

			// Inyectar el trackCode en el contexto usando la clave de response
			ctx := response.WithTrackCode(r.Context(), trackCode)

			// Envolver ResponseWriter para interceptar el código de estado HTTP
			lrw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default si no se llama a WriteHeader
			}

			// Ejecutar el pipeline de controladores
			next.ServeHTTP(lrw, r.WithContext(ctx))

			// Calcular latencia
			latency := time.Since(startTime).Milliseconds()

			// Obtener el código de respuesta (se inyecta mediante response.Success o response.Error)
			responseCode := lrw.Header().Get("X-Response-Code")
			if responseCode == "" {
				if lrw.statusCode >= 200 && lrw.statusCode < 300 {
					if lrw.statusCode == http.StatusCreated {
						responseCode = "CREATED"
					} else {
						responseCode = "SUCCESS"
					}
				} else {
					responseCode = "INTERNAL_ERROR"
				}
			}
			lrw.Header().Del("X-Response-Code")

			// Capturar el ID del usuario autenticado propagado mediante cabeceras internas
			var userID *int64
			if uIDStr := lrw.Header().Get("X-Authenticated-User-ID"); uIDStr != "" {
				var parsedID int64
				if _, err := fmt.Sscanf(uIDStr, "%d", &parsedID); err == nil {
					userID = &parsedID
				}
				lrw.Header().Del("X-Authenticated-User-ID")
			}

			// Filtrar logs innecesarios para evitar saturación de la base de datos (DB Bloat)
			shouldLog := true
			path := r.URL.Path

			// 1. Excluir lecturas de datos (GET/HEAD) y pings de monitoreo que no modifican estado
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				shouldLog = false
			}

			// 2. Excluir endpoints fuera de la API, Swagger, Docs y favicon
			if !strings.HasPrefix(path, "/api/") || 
			   strings.HasPrefix(path, "/swagger/") || 
			   strings.HasPrefix(path, "/docs/") || 
			   strings.Contains(path, "favicon.ico") || 
			   path == "/api/auth/public-key" {
				shouldLog = false
			}

			if shouldLog {
				// Registro asíncrono en base de datos mediante una goroutine dedicada
				go func(tCode string, uID *int64, method, path string, status int, respCode string, lat int64) {
					dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					query := `
						INSERT INTO api_logs (
							track_code, 
							user_id, 
							method, 
							path, 
							http_status, 
							response_code, 
							latency_ms
						)
						VALUES ($1, $2, $3, $4, $5, $6, $7);
					`
					_, err := db.Exec(dbCtx, query, tCode, uID, method, path, status, respCode, lat)
					if err != nil {
						slog.Error("Fallo al insertar log de auditoría en PostgreSQL", 
							"error", err, 
							"track_code", tCode,
						)
					}
				}(trackCode, userID, r.Method, path, lrw.statusCode, responseCode, latency)
			}
		})
	}
}

// AuthMiddleware intercepta las peticiones, valida el token JWT y guarda el ID del usuario en el contexto.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: cabecera Authorization ausente")
				return
			}

			// Validar formato "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: formato inválido (debe ser Bearer <token>)")
				return
			}

			tokenString := parts[1]

			// Parsear y validar firma del token
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				// Validar que se use el algoritmo HMAC esperado (HS256)
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritmo de firma inesperado: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: token expirado o inválido")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: payload de claims corrupto")
				return
			}

			// Extraer user_id (deserializa como float64 desde JSON estándar en JWT)
			userIDFloat, ok := claims["user_id"].(float64)
			if !ok {
				response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "autorización denegada: user_id ausente en claims")
				return
			}

			userID := int64(userIDFloat)

			// Propagar el ID del usuario en una cabecera interna temporal para que el middleware de trazabilidad la capture
			w.Header().Set("X-Authenticated-User-ID", fmt.Sprintf("%d", userID))

			// Inyectar el user_id en el contexto de manera segura
			ctx := context.WithValue(r.Context(), userContextKey, userID)

			// Continuar con el siguiente controlador
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext recupera el ID numérico del usuario desde el contexto del HTTP request.
func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userContextKey).(int64)
	return userID, ok
}
