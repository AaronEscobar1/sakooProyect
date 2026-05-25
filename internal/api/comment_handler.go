package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

// CommentRequest define los parámetros de entrada para agregar un comentario.
type CommentRequest struct {
	RateID  int64  `json:"rate_id"`
	Content string `json:"content"`
}

type CommentHandler struct {
	useCase domain.CommentUseCase
}

// NewCommentHandler crea un nuevo controlador HTTP de opiniones/comentarios.
func NewCommentHandler(useCase domain.CommentUseCase) *CommentHandler {
	return &CommentHandler{
		useCase: useCase,
	}
}

// HandleAddComment maneja POST /api/v1/rates/comments (JWT protegido)
// @Summary      Agregar comentario a una tasa
// @Description  Agrega una opinión o comentario de un usuario autenticado para una tasa de cambio específica.
// @Security     ApiKeyAuth
// @Tags         Comentarios
// @Accept       json
// @Produce      json
// @Param        body  body  CommentRequest  true  "Datos del comentario"
// @Success      200  {object}  response.APIResponse[domain.Comment]  "Comentario agregado exitosamente"
// @Failure      200  {object}  response.APIResponse[any]             "Error de validación o no autorizado"
// @Router       /api/v1/rates/comments [post]
func (h *CommentHandler) HandleAddComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "Usuario no autenticado")
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	if req.RateID <= 0 {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El ID de tasa 'rate_id' es requerido y debe ser mayor a cero")
		return
	}

	if req.Content == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El contenido del comentario no puede estar vacío")
		return
	}

	comment, err := h.useCase.AddComment(r.Context(), userID, req.RateID, req.Content)
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	response.Success(w, r.Context(), "CREATED", "Comentario agregado exitosamente", comment)
}

// HandleGetRateComments maneja GET /api/v1/rates/{rate_id}/comments (Público, opiniones del día)
// @Summary      Obtener opiniones del día para una tasa
// @Description  Retorna una lista de comentarios u opiniones realizadas durante el día para la tasa indicada.
// @Tags         Comentarios
// @Produce      json
// @Param        rate_id  path  int64  true  "ID de la tasa de cambio"
// @Success      200  {object}  response.APIResponse[[]domain.Comment]  "Comentarios del día obtenidos exitosamente"
// @Failure      200  {object}  response.APIResponse[any]               "ID de tasa inválido o error interno"
// @Router       /api/v1/rates/{rate_id}/comments [get]
func (h *CommentHandler) HandleGetRateComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere GET)")
		return
	}

	rateIDStr := r.PathValue("rate_id")
	if rateIDStr == "" {
		rateIDStr = r.URL.Query().Get("rate_id")
	}

	rateID, err := strconv.ParseInt(rateIDStr, 10, 64)
	if err != nil || rateID <= 0 {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "ID de tasa 'rate_id' inválido o ausente en la ruta")
		return
	}

	comments, err := h.useCase.GetCommentsByRate(r.Context(), rateID)
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Comentarios del día obtenidos exitosamente", comments)
}
