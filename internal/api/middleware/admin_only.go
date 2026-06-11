package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

// AdminOnly intercepta las peticiones a endpoints protegidos del BackOffice y verifica
// que el token JWT contenga el claim 'user_type' con el valor 'ADMIN'.
//
// Este middleware DEBE ejecutarse DESPUÉS de AuthMiddleware, que ya validó la firma,
// expiración y existencia de la sesión del token. AdminOnly solo se encarga de la
// verificación del rol/autorización.
//
// Si el token no contiene el claim 'user_type' o el valor no es 'ADMIN',
// responde con HTTP 403 Forbidden estandarizado.
func AdminOnly(jwtSecret string, userRepo domain.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Extraer el token JWT del header Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, r.Context(), http.StatusForbidden, "FORBIDDEN", "acceso denegado: permisos insuficientes para este recurso")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, r.Context(), http.StatusForbidden, "FORBIDDEN", "acceso denegado: permisos insuficientes para este recurso")
				return
			}

			tokenString := parts[1]

			// 2. Parsear el token para extraer claims (la firma ya fue validada por AuthMiddleware)
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritmo de firma inesperado: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				slog.Warn("AdminOnly: token JWT inválido o expirado detectado en capa de autorización")
				response.Error(w, r.Context(), http.StatusForbidden, "FORBIDDEN", "acceso denegado: permisos insuficientes para este recurso")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				slog.Warn("AdminOnly: claims del JWT corruptos")
				response.Error(w, r.Context(), http.StatusForbidden, "FORBIDDEN", "acceso denegado: permisos insuficientes para este recurso")
				return
			}

			// 3. Verificar el claim 'user_type'
			userType, exists := claims["user_type"].(string)
			if !exists || userType != "ADMIN" {
				slog.Warn("AdminOnly: intento de acceso a recurso administrativo denegado por rol insuficiente",
					"user_type", userType,
					"path", r.URL.Path,
					"method", r.Method,
				)
				response.Error(w, r.Context(), http.StatusForbidden, "FORBIDDEN", "acceso denegado: permisos insuficientes para este recurso")
				return
			}

			// 4. Refrescar de forma deslizante la expiración de la sesión en base de datos otros 10 minutos
			if err := userRepo.ExtendSession(r.Context(), tokenString, time.Now().Add(10*time.Minute)); err != nil {
				slog.Error("AdminOnly: no se pudo extender la expiración de la sesión en PostgreSQL", "error", err)
			}

			// 5. Autorización concedida: continuar con el handler siguiente
			next.ServeHTTP(w, r)
		})
	}
}
