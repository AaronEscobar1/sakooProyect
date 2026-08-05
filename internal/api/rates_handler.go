package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/aaron/sakoo-backend/internal/usecase"
	"github.com/shopspring/decimal"
)

type DashboardExchangeRate struct {
	RateID       int64  `json:"rate_id"`
	CurrencyCode string `json:"currency_code"`
	RateFrom     string `json:"rate_from"`
	RateTo       string `json:"rate_to"`
	RateAverage  string `json:"rate_average"`
	ValueDate    string `json:"value_date"`
}

type DashboardResponse struct {
	VariationPercent string                  `json:"variation_percent"`
	History          []DashboardExchangeRate `json:"history"`
}

type ConversionRequest struct {
	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
	Date     string          `json:"date,omitempty"`
}

type ConversionResponse struct {
	Currency        string `json:"currency"`
	OriginalAmount  string `json:"original_amount"`
	ConvertedAmount string `json:"converted_amount"`
}

type RatesHandler struct {
	dashboardUseCase  usecase.DashboardUseCase
	calculatorUseCase usecase.CalculatorUseCase
	ratesUseCase      *usecase.ExchangeRateUseCase
}

func NewRatesHandler(dashboardUseCase usecase.DashboardUseCase, calculatorUseCase usecase.CalculatorUseCase, ratesUseCase *usecase.ExchangeRateUseCase) *RatesHandler {
	return &RatesHandler{
		dashboardUseCase:  dashboardUseCase,
		calculatorUseCase: calculatorUseCase,
		ratesUseCase:      ratesUseCase,
	}
}

func (h *RatesHandler) HandleGetDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere GET)")
		return
	}

	currency := domain.CleanCurrencyCode(r.URL.Query().Get("currency"))
	if currency == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El parámetro query 'currency' es requerido")
		return
	}

	var refDate *time.Time
	if dateStr := r.URL.Query().Get("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El parámetro query 'date' debe tener el formato YYYY-MM-DD")
			return
		}
		refDate = &parsedDate
	}

	summary, err := h.dashboardUseCase.GetDashboardSummary(r.Context(), currency, refDate)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || ent.IsNotFound(err) {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "No se encontraron tasas de cambio para la moneda especificada")
			return
		}
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	var historyDTO []DashboardExchangeRate
	for _, rate := range summary.History {
		historyDTO = append(historyDTO, DashboardExchangeRate{
			RateID:       rate.ID,
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
		VariationPercent: summary.VariationPercent.String(),
		History:           historyDTO,
	}

	response.Success(w, r.Context(), "SUCCESS", "Resumen de dashboard obtenido exitosamente", res)
}

func (h *RatesHandler) HandleCalculateConversion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere POST)")
		return
	}

	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "Formato de cuerpo JSON inválido")
		return
	}

	req.Currency = domain.CleanCurrencyCode(req.Currency)
	if req.Currency == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El código de moneda 'currency' es requerido")
		return
	}

	if req.Amount.IsNegative() {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "El monto 'amount' no puede ser negativo")
		return
	}

	convertedAmount, err := h.calculatorUseCase.CalculateConversion(r.Context(), req.Currency, req.Amount, req.Date)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || ent.IsNotFound(err) {
			response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "No se encontraron tasas de cambio para realizar la conversión en la fecha indicada")
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

func (h *RatesHandler) HandleGetCalendarDates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, r.Context(), http.StatusOK, "METHOD_NOT_ALLOWED", "Método no permitido (se requiere GET)")
		return
	}

	dates, err := h.ratesUseCase.GetCalendarDates(r.Context())
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INTERNAL_ERROR", err.Error())
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "Fechas de calendario obtenidas exitosamente", dates)
}
