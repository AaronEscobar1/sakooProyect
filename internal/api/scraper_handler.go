package api

import (
	"fmt"
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScraperHandler expone los controladores HTTP del módulo de web scraping.
type ScraperHandler struct {
	bcvScraperUseCase       *usecase.ScraperUseCase
	mercantilScraperUseCase *usecase.MercantilScraperUseCase
	db                      *pgxpool.Pool
}

// NewScraperHandler crea una nueva instancia de ScraperHandler.
func NewScraperHandler(
	bcvScraperUseCase *usecase.ScraperUseCase, 
	mercantilScraperUseCase *usecase.MercantilScraperUseCase,
	db *pgxpool.Pool,
) *ScraperHandler {
	return &ScraperHandler{
		bcvScraperUseCase:       bcvScraperUseCase,
		mercantilScraperUseCase: mercantilScraperUseCase,
		db:                      db,
	}
}

// HandleScrapeNow fuerza y ejecuta la extracción de tasas en caliente de forma síncrona.
// @Summary      Forzar raspado de tasas BCV
// @Description  Ejecuta de forma síncrona el web scraper del Banco Central de Venezuela (BCV) para actualizar las tasas de cambio.
// @Security     ApiKeyAuth
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
// @Security     ApiKeyAuth
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

// HandleScrapeBinance fuerza y ejecuta el worker de Binance P2P en caliente de forma síncrona.
// @Summary      Forzar raspado de Binance P2P (USDT/USDC)
// @Description  Ejecuta de forma síncrona el worker de Binance P2P para un activo específico (USDT o USDC) y actualiza la base de datos.
// @Security     ApiKeyAuth
// @Tags         Administración
// @Param        asset  query     string  false  "Activo a raspar (USDT o USDC, por defecto USDT)"
// @Produce      json
// @Success      200  {object}  response.APIResponse[any]  "Raspado y actualización de Binance P2P completados con éxito"
// @Failure      400  {object}  response.APIResponse[any]  "Activo inválido"
// @Failure      500  {object}  response.APIResponse[any]  "Error interno al realizar el scraping"
// @Router       /api/admin/scrape-binance [post]
func (h *ScraperHandler) HandleScrapeBinance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	asset := r.URL.Query().Get("asset")
	if asset == "" {
		asset = "USDT"
	}

	if asset != "USDT" && asset != "USDC" {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_ASSET", "el activo solicitado debe ser USDT o USDC")
		return
	}

	err := scraper.RunBinanceWorker(r.Context(), h.db, asset)
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al ejecutar Binance P2P Worker: "+err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", fmt.Sprintf("Raspado y actualización de Binance P2P para %s ejecutado con éxito", asset), nil)
}
