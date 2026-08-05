package api

import (
	"fmt"
	"net/http"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/internal/infrastructure/scraper"
	"github.com/aaron/sakoo-backend/internal/usecase"
)

type ScraperHandler struct {
	bcvScraperUseCase *usecase.ScraperUseCase
	client            *ent.Client
}

func NewScraperHandler(
	bcvScraperUseCase *usecase.ScraperUseCase,
	client *ent.Client,
) *ScraperHandler {
	return &ScraperHandler{
		bcvScraperUseCase: bcvScraperUseCase,
		client:            client,
	}
}

func (h *ScraperHandler) HandleScrapeNow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	err := h.bcvScraperUseCase.ExecuteScraping(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "Error al realizar el scraping manual de tasas del BCV")
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Raspado y actualización de tasas de cambio del BCV ejecutado con éxito", nil)
}

func (h *ScraperHandler) HandleScrapeBinance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	asset := r.URL.Query().Get("asset")
	if asset == "" {
		asset = "USDT"
	}

	if asset != "USDT" && asset != "USDC" {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_ASSET", "El activo solicitado debe ser USDT o USDC")
		return
	}

	err := scraper.RunBinanceWorker(r.Context(), h.client, asset)
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "error al ejecutar Binance P2P Worker: "+err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", fmt.Sprintf("Raspado y actualización de Binance P2P para %s ejecutado con éxito", asset), nil)
}
