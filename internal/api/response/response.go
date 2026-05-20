package response

import (
	"context"
	"encoding/json"
	"net/http"
)

// contextKey es un tipo de clave privada para evitar colisiones en context.Context
type contextKey struct{}

// TrackCodeKey es la clave utilizada para almacenar y recuperar el TrackCode en el contexto del request
var TrackCodeKey = contextKey{}

// APIResponse define el formato estructurado único y estandarizado para todas las respuestas de la API con tipado fuerte (Genéricos)
type APIResponse[T any] struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      T      `json:"data,omitempty"`
	TrackCode string `json:"track_code"`
}

// GetTrackCode extrae el código de seguimiento (Trace ID) desde el context.Context
func GetTrackCode(ctx context.Context) string {
	if val, ok := ctx.Value(TrackCodeKey).(string); ok {
		return val
	}
	return ""
}

// WithTrackCode inyecta un código de seguimiento (Trace ID) en el context.Context
func WithTrackCode(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, TrackCodeKey, code)
}

// GetResponseCodeInt mapea el código semántico interno a su código numérico correspondiente.
// Retorna 1000 para casos correctos (SUCCESS, CREATED) y códigos diferentes para casos incorrectos.
func GetResponseCodeInt(code string) int {
	switch code {
	case "SUCCESS", "CREATED":
		return 1000
	case "INVALID_JSON":
		return 1001
	case "BAD_REQUEST":
		return 1002
	case "UNAUTHORIZED":
		return 1003
	case "USER_ALREADY_EXISTS":
		return 1004
	case "INTERNAL_ERROR":
		return 1005
	case "METHOD_NOT_ALLOWED":
		return 1006
	default:
		return 9999 // Error genérico/desconocido
	}
}

// Success envía una respuesta HTTP exitosa (JSON) estandarizada e inyecta el track_code y X-Response-Code
func Success(w http.ResponseWriter, ctx context.Context, code string, msg string, data any) {
	trackCode := GetTrackCode(ctx)

	// Inyectar el código de respuesta en las cabeceras para que el middleware de logs lo intercepte
	w.Header().Set("X-Response-Code", code)
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	resp := APIResponse[any]{
		Code:      GetResponseCodeInt(code),
		Message:   msg,
		Data:      data,
		TrackCode: trackCode,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Error envía una respuesta HTTP fallida (JSON) estandarizada e inyecta el track_code y X-Response-Code
func Error(w http.ResponseWriter, ctx context.Context, httpStatus int, code string, msg string) {
	trackCode := GetTrackCode(ctx)

	// Inyectar el código de respuesta en las cabeceras para que el middleware de logs lo intercepte
	w.Header().Set("X-Response-Code", code)
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	resp := APIResponse[any]{
		Code:      GetResponseCodeInt(code),
		Message:   msg,
		TrackCode: trackCode,
	}

	_ = json.NewEncoder(w).Encode(resp)
}
