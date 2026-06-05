package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/internal/usecase"
)

// RateResponse define el DTO de respuesta para una tasa de cambio individual.
type RateResponse struct {
	RateID       int64  `json:"rate_id"`
	CurrencyCode string `json:"currency_code"`
	RateFrom     string `json:"rate_from"`
	RateTo       string `json:"rate_to"`
	RateAverage  string `json:"rate_average"`
	ValueDate    string `json:"value_date"`
	UpdatedAt    string `json:"updated_at"`
}

// HistoryRequest define los parámetros de consulta para el historial de tasas.
type HistoryRequest struct {
	Page         int    `json:"page"`
	Limit        int    `json:"limit"`
	CurrencyCode string `json:"currency_code,omitempty"`
	StartDate    string `json:"start_date,omitempty"` // YYYY-MM-DD
	EndDate      string `json:"end_date,omitempty"`   // YYYY-MM-DD
}

// PaginatedRatesResponse define el formato DTO de salida paginado para el historial de tasas.
type PaginatedRatesResponse struct {
	Items       []RateResponse `json:"items"`
	TotalItems  int            `json:"total_items"`
	TotalPages  int            `json:"total_pages"`
	CurrentPage int            `json:"current_page"`
	Limit       int            `json:"limit"`
}

// ExchangeRateHandler expone los controladores HTTP de tasas de cambio.
type ExchangeRateHandler struct {
	useCase *usecase.ExchangeRateUseCase
}

// NewExchangeRateHandler crea una nueva instancia de ExchangeRateHandler.
func NewExchangeRateHandler(useCase *usecase.ExchangeRateUseCase) *ExchangeRateHandler {
	return &ExchangeRateHandler{
		useCase: useCase,
	}
}

// HandleGetLatestRates retorna las tasas más recientes de las monedas, expuesto de forma pública.
// @Summary      Obtener tasas de cambio más recientes
// @Description  Retorna las últimas tasas de cambio obtenidas para todas las monedas registradas.
// @Tags         Tasas de Cambio
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]RateResponse]  "Tasas obtenidas exitosamente"
// @Failure      500  {object}  response.APIResponse[any]            "Error interno al obtener tasas"
// @Router       /api/rates [post]
func (h *ExchangeRateHandler) HandleGetLatestRates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	rates, err := h.useCase.GetLatestRates(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusInternalServerError, "INTERNAL_ERROR", "Error al obtener las tasas de cambio")
		return
	}

	var data []RateResponse
	for _, rate := range rates {
		data = append(data, RateResponse{
			RateID:       rate.ID,
			CurrencyCode: rate.CurrencyCode,
			RateFrom:     rate.RateFrom.String(),
			RateTo:       rate.RateTo.String(),
			RateAverage:  rate.RateAverage.String(),
			ValueDate:    rate.ValueDate.Format("2006-01-02"),
			UpdatedAt:    rate.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	// Si no hay tasas, devolvemos un slice vacío en vez de null
	if data == nil {
		data = []RateResponse{}
	}

	response.Success(w, r.Context(), "SUCCESS", "Tasas de cambio obtenidas exitosamente", data)
}

// HandleGetRatesHistory maneja la petición POST /api/rates/history para listar el historial de tasas con paginado y filtros.
// @Summary      Obtener historial de tasas de cambio
// @Description  Retorna una lista paginada del historial de tasas de cambio según moneda y rangos de fechas opcionales.
// @Tags         Tasas de Cambio
// @Accept       json
// @Produce      json
// @Param        body  body  HistoryRequest  true  "Parámetros de paginación y filtros"
// @Success      200  {object}  response.APIResponse[PaginatedRatesResponse]  "Historial obtenido exitosamente"
// @Failure      400  {object}  response.APIResponse[any]                     "Datos de entrada inválidos"
// @Router       /api/rates/history [post]
func (h *ExchangeRateHandler) HandleGetRatesHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	var req HistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	// Valores por defecto si no vienen especificados
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 {
		req.Limit = 10
	}

	rates, totalItems, err := h.useCase.GetRatesHistory(
		r.Context(),
		req.Page,
		req.Limit,
		req.CurrencyCode,
		req.StartDate,
		req.EndDate,
	)
	if err != nil {
		response.Error(w, r.Context(), http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	var data []RateResponse
	for _, rate := range rates {
		data = append(data, RateResponse{
			RateID:       rate.ID,
			CurrencyCode: rate.CurrencyCode,
			RateFrom:     rate.RateFrom.String(),
			RateTo:       rate.RateTo.String(),
			RateAverage:  rate.RateAverage.String(),
			ValueDate:    rate.ValueDate.Format("2006-01-02"),
			UpdatedAt:    rate.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	if data == nil {
		data = []RateResponse{}
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = (totalItems + req.Limit - 1) / req.Limit
	}

	res := PaginatedRatesResponse{
		Items:       data,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: req.Page,
		Limit:       req.Limit,
	}

	response.Success(w, r.Context(), "SUCCESS", "Historial de tasas de cambio obtenido exitosamente", res)
}
