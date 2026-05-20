package api

import (
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type BannerHandler struct {
	useCase domain.BannerUseCase
}

// NewBannerHandler crea un nuevo controlador HTTP para la gestión de banners publicitarios.
func NewBannerHandler(useCase domain.BannerUseCase) *BannerHandler {
	return &BannerHandler{
		useCase: useCase,
	}
}

// HandleGetBanners maneja GET /api/v1/banners (Público, retorna banners activos)
// @Summary      Obtener banners activos
// @Description  Retorna una lista de banners publicitarios activos en el sistema.
// @Tags         Banners
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]domain.Banner]  "Banners activos recuperados exitosamente"
// @Failure      200  {object}  response.APIResponse[any]              "Error interno del servidor"
// @Router       /api/v1/banners [get]
func (h *BannerHandler) HandleGetBanners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	banners, err := h.useCase.GetActiveBanners(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Banners activos recuperados exitosamente", banners)
}
