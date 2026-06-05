package api

import (
	"github.com/AaronEscobar1/common/middleware"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type NotificationHandler struct {
	useCase domain.NotificationUseCase
}

// NewNotificationHandler crea un nuevo controlador para notificaciones push.
func NewNotificationHandler(useCase domain.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{
		useCase: useCase,
	}
}

// HandleRegisterDevice maneja POST /api/v1/devices/register
// @Summary      Registrar dispositivo para push
// @Description  Registra un FCM Token de Firebase del dispositivo del usuario autenticado para recibir notificaciones push.
// @Tags         Notificaciones
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        body  body  domain.RegisterDeviceRequest  true  "Datos del dispositivo"
// @Success      200   {object}  response.APIResponse[any]  "Dispositivo registrado con éxito"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Router       /api/v1/devices/register [post]
func (h *NotificationHandler) HandleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "Autorización denegada: usuario no autenticado")
		return
	}

	var req domain.RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	if err := h.useCase.RegisterDevice(r.Context(), userID, req.Token, req.Platform); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Dispositivo registrado exitosamente para recibir notificaciones push", nil)
}

// HandleUnregisterDevice maneja POST /api/v1/devices/unregister
// @Summary      Dar de baja un dispositivo
// @Description  Elimina el FCM Token del dispositivo del usuario autenticado, deteniendo el envío de notificaciones.
// @Tags         Notificaciones
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        body  body  domain.UnregisterDeviceRequest  true  "Token del dispositivo"
// @Success      200   {object}  response.APIResponse[any]  "Dispositivo dado de baja con éxito"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Router       /api/v1/devices/unregister [post]
func (h *NotificationHandler) HandleUnregisterDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "Autorización denegada: usuario no autenticado")
		return
	}

	var req domain.UnregisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	if err := h.useCase.UnregisterDevice(r.Context(), userID, req.Token); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Dispositivo dado de baja exitosamente en el sistema", nil)
}

// HandleGetNotifications maneja GET /api/v1/notifications
// @Summary      Obtener bandeja de entrada (Inbox)
// @Description  Retorna la lista de notificaciones históricas recibidas por el usuario autenticado.
// @Tags         Notificaciones
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200   {object}  response.APIResponse[[]domain.Notification]  "Historial de notificaciones"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Router       /api/v1/notifications [get]
func (h *NotificationHandler) HandleGetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere GET)")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "Autorización denegada: usuario no autenticado")
		return
	}

	list, err := h.useCase.GetNotifications(r.Context(), userID)
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Bandeja de notificaciones obtenida con éxito", list)
}

// HandleMarkAsRead maneja PUT /api/v1/notifications/{id}/read
// @Summary      Marcar notificación como leída
// @Description  Actualiza el estado de una notificación en el inbox para marcarla como leída.
// @Tags         Notificaciones
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id    path      int  true  "ID de la notificación"
// @Success      200   {object}  response.APIResponse[any]  "Notificación marcada como leída"
// @Failure      401   {object}  response.APIResponse[any]  "No autorizado"
// @Router       /api/v1/notifications/{id}/read [put]
func (h *NotificationHandler) HandleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere PUT)")
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusUnauthorized, "UNAUTHORIZED", "Autorización denegada: usuario no autenticado")
		return
	}

	// Extraer ID de la notificación del path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "ID de notificación ausente en la ruta")
		return
	}
	idStr := parts[4] // /api/v1/notifications/{id}/read -> index 4
	notifID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "ID de notificación inválido")
		return
	}

	if err := h.useCase.MarkNotificationAsRead(r.Context(), userID, notifID); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Notificación marcada como leída exitosamente", nil)
}

// HandleSendAdminNotification maneja POST /api/admin/notifications/send
// @Summary      Enviar notificación administrativa (BackOffice)
// @Description  Envía una notificación push asíncrona a un usuario en particular o a todos los usuarios del sistema (broadcast) con un título, cuerpo y payload de datos.
// @Security     ApiKeyAuth
// @Tags         Notificaciones
// @Accept       json
// @Produce      json
// @Param        body  body  domain.SendAdminNotificationRequest  true  "Datos de la notificación"
// @Success      200   {object}  response.APIResponse[any]  "Notificación procesada y enviada"
// @Router       /api/admin/notifications/send [post]
func (h *NotificationHandler) HandleSendAdminNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	var req domain.SendAdminNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	if req.Title == "" || req.Body == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "Título y cuerpo de notificación son requeridos")
		return
	}

	if req.UserID != nil {
		// Envío individual
		if err := h.useCase.SendSystemNotification(r.Context(), *req.UserID, req.Title, req.Body, req.Data); err != nil {
			response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
	} else {
		// Envío broadcast a todos
		if err := h.useCase.SendBroadcastNotification(r.Context(), req.Title, req.Body, req.Data); err != nil {
			response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
	}

	response.Success(w, r.Context(), "SUCCESS", "Notificación administrativa enviada y procesada correctamente", nil)
}

// HandleTestPushNotification maneja POST /api/admin/notifications/test
// @Summary      Probar envío de notificaciones push por moneda
// @Description  Simula el cambio de tasa de una moneda específica y gatilla su respectiva notificación push FCM al canal "exchange_rates".
// @Tags         Notificaciones
// @Produce      json
// @Param        currency  query     string  true  "Código de la moneda o fuente para gatillar la notificación (valores soportados: USDT, USDC, UDI, MERCANTIL, USD, EUR, BCV)"
// @Success      200       {object}  response.APIResponse[any]  "Notificación de prueba enviada con éxito"
// @Failure      400       {object}  response.APIResponse[any]  "Moneda inválida o no soportada"
// @Failure      500       {object}  response.APIResponse[any]  "Error al enviar la notificación de prueba"
// @Router       /api/admin/notifications/test [post]
func (h *NotificationHandler) HandleTestPushNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	currency := strings.ToUpper(r.URL.Query().Get("currency"))
	if currency == "" {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", "El parámetro query 'currency' es requerido")
		return
	}

	var title, body string
	payload := make(map[string]interface{})

	switch currency {
	case "USDT", "USDC":
		title = fmt.Sprintf("¡La tasa de %s (Binance P2P) ha cambiado! 🚀", currency)
		body = "La nueva tasa promedio de Binance P2P es de 39.75 Bs."
		payload = map[string]interface{}{
			"type":          "rate_update",
			"source":        "BINANCE",
			"currency_code": currency,
			"rate":          "39.75",
		}
	case "UDI", "MERCANTIL":
		title = "¡La tasa de Mercantil ha cambiado! 🚀"
		body = "La nueva tasa del Dólar Intervención es de 39.50 Bs."
		payload = map[string]interface{}{
			"type":          "rate_update",
			"source":        "MERCANTIL",
			"currency_code": "UDI",
			"rate":          "39.50",
		}
	case "USD":
		title = "¡La tasa de USD ha cambiado! 🚀"
		body = "La nueva tasa oficial es de 36.50 Bs."
		payload = map[string]interface{}{
			"type":          "rate_update",
			"source":        "BCV",
			"currency_code": "USD",
			"rate":          "36.50",
		}
	case "EUR":
		title = "¡La tasa de EUR ha cambiado! 🚀"
		body = "La nueva tasa oficial es de 39.40 Bs."
		payload = map[string]interface{}{
			"type":          "rate_update",
			"source":        "BCV",
			"currency_code": "EUR",
			"rate":          "39.40",
		}
	case "BCV":
		title = "¡Las tasas del BCV han cambiado! 🚀"
		body = "El Banco Central de Venezuela actualizó múltiples tasas oficiales en la plataforma."
		payload = map[string]interface{}{
			"type":   "rate_update",
			"source": "BCV",
		}
	default:
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_CURRENCY", "Moneda no soportada. Use: USDT, USDC, UDI, MERCANTIL, USD, EUR, BCV")
		return
	}

	err := h.useCase.SendTopicNotification(r.Context(), "exchange_rates", title, body, payload)
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "Error al enviar la notificación push de prueba: "+err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", fmt.Sprintf("Notificación push de prueba para %s enviada con éxito al topic exchange_rates", currency), nil)
}
