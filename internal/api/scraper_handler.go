package api

import (
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/usecase"
)

// ScraperHandler expone los controladores HTTP del módulo de web scraping.
type ScraperHandler struct {
	bcvScraperUseCase       *usecase.ScraperUseCase
	mercantilScraperUseCase *usecase.MercantilScraperUseCase
}

// NewScraperHandler crea una nueva instancia de ScraperHandler.
func NewScraperHandler(bcvScraperUseCase *usecase.ScraperUseCase, mercantilScraperUseCase *usecase.MercantilScraperUseCase) *ScraperHandler {
	return &ScraperHandler{
		bcvScraperUseCase:       bcvScraperUseCase,
		mercantilScraperUseCase: mercantilScraperUseCase,
	}
}

// HandleScrapeNow fuerza y ejecuta la extracción de tasas en caliente de forma síncrona.
// @Summary      Forzar raspado de tasas BCV
// @Description  Ejecuta de forma síncrona el web scraper del Banco Central de Venezuela (BCV) para actualizar las tasas de cambio.
// @Tags         Administración
// @Produce      json
// @Success      200  {object}  response.APIResponse[any]  "Raspado y actualización del BCV ejecutados con éxito"
// @Failure      500  {object}  response.APIResponse[any]  "Error interno al realizar el scraping"
// @Router       /api/admin/scrape-now [post]
func (h *ScraperHandler) HandleScrapeNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	err := h.bcvScraperUseCase.ExecuteScraping(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al realizar el scraping manual de tasas del BCV")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Raspado y actualización de tasas de cambio del BCV ejecutado con éxito", nil)
}

// HandleScrapeMercantilNow fuerza y ejecuta la extracción de tasas de Mercantil en caliente de forma síncrona.
// @Summary      Forzar raspado de tasas Mercantil
// @Description  Ejecuta de forma síncrona el web scraper de Mercantil para actualizar las tasas de cambio de este banco.
// @Tags         Administración
// @Produce      json
// @Success      200  {object}  response.APIResponse[any]  "Raspado y actualización de Mercantil ejecutados con éxito"
// @Failure      500  {object}  response.APIResponse[any]  "Error interno al realizar el scraping"
// @Router       /api/admin/scrape-mercantil [post]
func (h *ScraperHandler) HandleScrapeMercantilNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	err := h.mercantilScraperUseCase.ExecuteScraping(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al realizar el scraping manual de tasas de Mercantil")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Raspado y actualización de tasas de cambio de Mercantil ejecutado con éxito", nil)
}
