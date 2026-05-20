package api

import (
	"encoding/json"
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

// SendMessageRequest define el DTO de entrada para enviar un mensaje.
type SendMessageRequest struct {
	ReceiverID int64  `json:"receiver_id"`
	Content    string `json:"content"`
}

type MessageHandler struct {
	useCase domain.MessageUseCase
}

// NewMessageHandler crea un nuevo controlador de mensajería interna.
func NewMessageHandler(useCase domain.MessageUseCase) *MessageHandler {
	return &MessageHandler{
		useCase: useCase,
	}
}

// HandleSendMessage maneja POST /api/v1/messages/send
// @Summary      Enviar un mensaje interno
// @Description  Envía un mensaje de texto de parte del usuario autenticado a otro usuario registrado en el sistema.
// @Security     ApiKeyAuth
// @Tags         Mensajería
// @Accept       json
// @Produce      json
// @Param        body  body  SendMessageRequest  true  "Datos del mensaje a enviar"
// @Success      200  {object}  response.APIResponse[domain.Message]  "Mensaje enviado exitosamente"
// @Failure      200  {object}  response.APIResponse[any]             "Destinatario inválido, contenido vacío o no autorizado"
// @Router       /api/v1/messages/send [post]
func (h *MessageHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	senderID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "usuario no autenticado")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.ReceiverID <= 0 {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el destinatario 'receiver_id' es requerido y debe ser mayor a cero")
		return
	}

	if req.Content == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el contenido 'content' no puede estar vacío")
		return
	}

	msg, err := h.useCase.SendMessage(r.Context(), senderID, req.ReceiverID, req.Content)
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Mensaje enviado exitosamente", msg)
}

// HandleGetMessages maneja GET /api/v1/messages
// @Summary      Obtener mensajes recibidos y enviados
// @Description  Obtiene la lista completa de mensajes de mensajería interna (tanto recibidos como enviados) del usuario autenticado.
// @Security     ApiKeyAuth
// @Tags         Mensajería
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]domain.Message]  "Mensajes obtenidos exitosamente"
// @Failure      200  {object}  response.APIResponse[any]               "No autorizado o error interno"
// @Router       /api/v1/messages [get]
func (h *MessageHandler) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "usuario no autenticado")
		return
	}

	messages, err := h.useCase.GetMessages(r.Context(), userID)
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Mensajes obtenidos exitosamente", messages)
}

// HandleGetUnreadCount maneja GET /api/v1/messages/unread-count
// @Summary      Obtener cantidad de mensajes no leídos
// @Description  Retorna el conteo total de mensajes recibidos por el usuario autenticado que aún no han sido marcados como leídos.
// @Security     ApiKeyAuth
// @Tags         Mensajería
// @Produce      json
// @Success      200  {object}  response.APIResponse[int]  "Conteo obtenido exitosamente"
// @Failure      200  {object}  response.APIResponse[any]  "No autorizado o error interno"
// @Router       /api/v1/messages/unread-count [get]
func (h *MessageHandler) HandleGetUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "usuario no autenticado")
		return
	}

	count, err := h.useCase.GetUnreadCount(r.Context(), userID)
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Mensajes no leídos contados exitosamente", count)
}
