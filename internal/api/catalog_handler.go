package api

import (
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/domain"
)

type CatalogHandler struct {
	useCase domain.CatalogUseCase
}

// NewCatalogHandler crea un nuevo controlador HTTP para la gestión de catálogos.
func NewCatalogHandler(useCase domain.CatalogUseCase) *CatalogHandler {
	return &CatalogHandler{
		useCase: useCase,
	}
}

// HandleGetDocumentTypes maneja GET /api/v1/catalogs/document-types (Público, retorna catálogo de tipos de documento)
// @Summary      Obtener catálogo de tipos de documento
// @Description  Retorna una lista de tipos de documento de identidad registrados en el sistema.
// @Tags         Catálogos
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]domain.DocumentType]  "Tipos de documento recuperados exitosamente"
// @Failure      200  {object}  response.APIResponse[any]                    "Error interno del servidor"
// @Router       /api/v1/catalogs/document-types [get]
func (h *CatalogHandler) HandleGetDocumentTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	docTypes, err := h.useCase.GetDocumentTypes(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Tipos de documento recuperados exitosamente", docTypes)
}

// HandleGetCurrencies maneja GET /api/v1/catalogs/currencies (Público, retorna catálogo de monedas)
// @Summary      Obtener catálogo de monedas
// @Description  Retorna una lista de monedas registradas en el sistema.
// @Tags         Catálogos
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]domain.Currency]  "Monedas recuperadas exitosamente"
// @Failure      200  {object}  response.APIResponse[any]                "Error interno del servidor"
// @Router       /api/v1/catalogs/currencies [get]
func (h *CatalogHandler) HandleGetCurrencies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	currencies, err := h.useCase.GetCurrencies(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Monedas recuperadas exitosamente", currencies)
}
