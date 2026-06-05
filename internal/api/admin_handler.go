package api

import (
	"net/http"
	"strconv"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

// AdminHandler expone los controladores de endpoints administrativos de Sakoo.
type AdminHandler struct {
	telemetryUseCase domain.TelemetryUseCase
}

// NewAdminHandler crea una nueva instancia de AdminHandler.
func NewAdminHandler(telemetryUseCase domain.TelemetryUseCase) *AdminHandler {
	return &AdminHandler{
		telemetryUseCase: telemetryUseCase,
	}
}

// PaginatedLogsResponse representa el DTO de respuesta paginado para los logs de auditoría.
type PaginatedLogsResponse struct {
	Items       []domain.APILog `json:"items"`
	TotalItems  int             `json:"total_items"`
	TotalPages  int             `json:"total_pages"`
	CurrentPage int             `json:"current_page"`
	Limit       int             `json:"limit"`
}

// HandleGetAuditLogs maneja GET /api/admin/logs (Protegido por X-Admin-Api-Key)
// @Summary      Obtener logs de auditoría
// @Description  Retorna un listado paginado y ordenado de los logs de auditoría registrados para operaciones mutables de la API.
// @Security     ApiKeyAuth
// @Tags         Administración
// @Produce      json
// @Param        page   query  int  false  "Número de página (por defecto 1)"
// @Param        limit  query  int  false  "Límite de registros por página (por defecto 10)"
// @Success      200    {object}  response.APIResponse[PaginatedLogsResponse]  "Historial de logs de auditoría obtenido exitosamente"
// @Failure      200    {object}  response.APIResponse[any]                    "Error de validación o no autorizado"
// @Router       /api/admin/logs [get]
func (h *AdminHandler) HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere GET)")
		return
	}

	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, totalItems, err := h.telemetryUseCase.GetAPILogs(r.Context(), page, limit)
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "Error al recuperar logs de auditoría")
		return
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + limit - 1) / limit
	}

	res := PaginatedLogsResponse{
		Items:       logs,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: page,
		Limit:       limit,
	}

	response.Success(w, r.Context(), "SUCCESS", "Historial de logs de auditoría obtenido exitosamente", res)
}
