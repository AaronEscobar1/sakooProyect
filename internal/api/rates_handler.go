package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aaron/sakoo-backend/internal/api/response"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

// DashboardExchangeRate representa el DTO de una tasa en el dashboard.
type DashboardExchangeRate struct {
	CurrencyCode string `json:"currency_code"`
	RateFrom     string `json:"rate_from"`
	RateTo       string `json:"rate_to"`
	RateAverage  string `json:"rate_average"`
	ValueDate    string `json:"value_date"`
}

// DashboardResponse define la estructura del resumen del dashboard.
type DashboardResponse struct {
	LatestRate       DashboardExchangeRate   `json:"latest_rate"`
	VariationPercent string                  `json:"variation_percent"`
	History          []DashboardExchangeRate `json:"history"`
}

// ConversionRequest define el DTO de entrada para la calculadora de conversión.
type ConversionRequest struct {
	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
	Date     string          `json:"date,omitempty"` // Opcional para calculadora retroactiva
}

// ConversionResponse define el DTO de salida para el resultado de conversión.
type ConversionResponse struct {
	Currency        string `json:"currency"`
	OriginalAmount  string `json:"original_amount"`
	ConvertedAmount string `json:"converted_amount"`
}

// RatesHandler maneja las peticiones HTTP del Core Business (Dashboard y Calculadora).
type RatesHandler struct {
	dashboardUseCase  usecase.DashboardUseCase
	calculatorUseCase usecase.CalculatorUseCase
	ratesUseCase      *usecase.ExchangeRateUseCase
}

// NewRatesHandler crea una nueva instancia de RatesHandler.
func NewRatesHandler(dashboardUseCase usecase.DashboardUseCase, calculatorUseCase usecase.CalculatorUseCase, ratesUseCase *usecase.ExchangeRateUseCase) *RatesHandler {
	return &RatesHandler{
		dashboardUseCase:  dashboardUseCase,
		calculatorUseCase: calculatorUseCase,
		ratesUseCase:      ratesUseCase,
	}
}

// HandleGetDashboardSummary maneja GET /api/v1/rates/dashboard
// @Summary      Obtener resumen del dashboard
// @Description  Retorna la última tasa de la moneda especificada, su porcentaje de variación y su historial de tasas recientes.
// @Tags         Core Business
// @Produce      json
// @Param        currency  query  string  true  "Código de la moneda (ej. USD)"
// @Success      200  {object}  response.APIResponse[DashboardResponse]  "Resumen de dashboard obtenido exitosamente"
// @Failure      200  {object}  response.APIResponse[any]                "Moneda no encontrada o error interno"
// @Router       /api/v1/rates/dashboard [get]
func (h *RatesHandler) HandleGetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	currency := r.URL.Query().Get("currency")
	if currency == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el parámetro query 'currency' es requerido")
		return
	}

	summary, err := h.dashboardUseCase.GetDashboardSummary(r.Context(), currency)
	if err != nil {
		// Control estricto de pgx.ErrNoRows: Responder HTTP 200, código interno diferente de 1000
		if errors.Is(err, pgx.ErrNoRows) || (err != nil && (errors.Is(err, pgx.ErrNoRows) || (errors.Unwrap(err) != nil && errors.Is(errors.Unwrap(err), pgx.ErrNoRows)) || err.Error() == "no rows in result set" || (len(err.Error()) > 20 && err.Error()[len(err.Error())-22:] == "no rows in result set"))) {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "no se encontraron tasas de cambio para la moneda especificada")
			return
		}
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	var historyDTO []DashboardExchangeRate
	for _, rate := range summary.History {
		historyDTO = append(historyDTO, DashboardExchangeRate{
			CurrencyCode: rate.CurrencyCode,
			RateFrom:     rate.RateFrom.String(),
			RateTo:       rate.RateTo.String(),
			RateAverage:  rate.RateAverage.String(),
			ValueDate:    rate.ValueDate.Format("2006-01-02"),
		})
	}
	if historyDTO == nil {
		historyDTO = []DashboardExchangeRate{}
	}

	res := DashboardResponse{
		LatestRate: DashboardExchangeRate{
			CurrencyCode: summary.LatestRate.CurrencyCode,
			RateFrom:     summary.LatestRate.RateFrom.String(),
			RateTo:       summary.LatestRate.RateTo.String(),
			RateAverage:  summary.LatestRate.RateAverage.String(),
			ValueDate:    summary.LatestRate.ValueDate.Format("2006-01-02"),
		},
		VariationPercent: summary.VariationPercent.String(),
		History:           historyDTO,
	}

	response.Success(w, r.Context(), "SUCCESS", "Resumen de dashboard obtenido exitosamente", res)
}

// HandleCalculateConversion maneja POST /api/v1/rates/calculate
// @Summary      Calcular conversión de monedas
// @Description  Calcula la conversión de un monto específico a la moneda objetivo según tasas vigentes o históricas.
// @Tags         Core Business
// @Accept       json
// @Produce      json
// @Param        body  body  ConversionRequest  true  "Datos para la conversión"
// @Success      200  {object}  response.APIResponse[ConversionResponse]  "Conversión realizada exitosamente"
// @Failure      200  {object}  response.APIResponse[any]                 "Moneda no encontrada o monto negativo"
// @Router       /api/v1/rates/calculate [post]
func (h *RatesHandler) HandleCalculateConversion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere POST)")
		return
	}

	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "formato de cuerpo JSON inválido")
		return
	}

	if req.Currency == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el código de moneda 'currency' es requerido")
		return
	}

	if req.Amount.IsNegative() {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el monto 'amount' no puede ser negativo")
		return
	}

	convertedAmount, err := h.calculatorUseCase.CalculateConversion(r.Context(), req.Currency, req.Amount, req.Date)
	if err != nil {
		// Control estricto de pgx.ErrNoRows: Responder HTTP 200, código interno diferente de 1000
		if errors.Is(err, pgx.ErrNoRows) || (err != nil && (errors.Is(err, pgx.ErrNoRows) || (errors.Unwrap(err) != nil && errors.Is(errors.Unwrap(err), pgx.ErrNoRows)) || err.Error() == "no rows in result set" || (len(err.Error()) > 20 && err.Error()[len(err.Error())-22:] == "no rows in result set"))) {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "no se encontraron tasas de cambio para realizar la conversión en la fecha indicada")
			return
		}
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	res := ConversionResponse{
		Currency:        req.Currency,
		OriginalAmount:  req.Amount.String(),
		ConvertedAmount: convertedAmount.String(),
	}

	response.Success(w, r.Context(), "SUCCESS", "Conversión realizada exitosamente", res)
}

// HandleGetCalendarDates maneja GET /api/v1/rates/calendar-dates
// @Summary      Obtener fechas con tasas registradas
// @Description  Retorna una lista de todas las fechas que tienen tasas registradas en el sistema.
// @Tags         Core Business
// @Produce      json
// @Success      200  {object}  response.APIResponse[[]string]  "Fechas obtenidas exitosamente"
// @Failure      200  {object}  response.APIResponse[any]       "Error interno"
// @Router       /api/v1/rates/calendar-dates [get]
func (h *RatesHandler) HandleGetCalendarDates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "método no permitido (se requiere GET)")
		return
	}

	dates, err := h.ratesUseCase.GetCalendarDates(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Fechas de calendario obtenidas exitosamente", dates)
}
