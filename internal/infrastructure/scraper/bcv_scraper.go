package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

// ScraperService define el contrato para el servicio de raspado de tasas de cambio.
type ScraperService interface {
	ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error)
}

// BCVScraper implementa ScraperService para obtener tasas de cambio oficiales del BCV
// utilizando la API pública de ve.dolarapi.com como fuente de datos, la cual
// no está sujeta al bloqueo de IPs de datacenters (Cloudflare) que aplica el sitio web del BCV.
type BCVScraper struct {
	apiURL string
}

// dolarAPIResponse representa un item de la respuesta de ve.dolarapi.com
type dolarAPIResponse struct {
	Moneda              string   `json:"moneda"`
	Fuente              string   `json:"fuente"`
	Nombre              string   `json:"nombre"`
	Compra              *float64 `json:"compra"`
	Venta               *float64 `json:"venta"`
	Promedio            float64  `json:"promedio"`
	FechaActualizacion  string   `json:"fechaActualizacion"`
}

// NewBCVScraper crea e inicializa una nueva instancia de BCVScraper usando ve.dolarapi.com.
func NewBCVScraper() ScraperService {
	return &BCVScraper{
		apiURL: "https://ve.dolarapi.com/v1/dolares",
	}
}

// ScrapeRates obtiene las tasas oficiales del BCV para USD y EUR desde ve.dolarapi.com.
// Esta API actúa como mirror del BCV oficial sin bloqueos de IP de datacenter.
func (s *BCVScraper) ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	slog.Info("Iniciando obtención de tasas del BCV vía ve.dolarapi.com...", "url", s.apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error al construir request a dolarapi: %w", err)
	}
	req.Header.Set("User-Agent", "SakooBackend/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al consultar ve.dolarapi.com: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ve.dolarapi.com respondió con status %d", resp.StatusCode)
	}

	var apiResp []dolarAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("error al decodificar respuesta de dolarapi: %w", err)
	}

	if len(apiResp) == 0 {
		return nil, fmt.Errorf("ve.dolarapi.com retornó una lista vacía de monedas")
	}

	// Fecha valor: hora Venezuela (UTC-4), truncada al día
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	defaultValueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	var rates []domain.ExchangeRate

	for _, item := range apiResp {
		// Solo procesar tasas de fuente "oficial" (BCV)
		if item.Fuente != "oficial" {
			continue
		}

		if item.Promedio <= 0 {
			slog.Warn("Tasa oficial con promedio inválido ignorada", "moneda", item.Moneda)
			continue
		}

		// Determinar la fecha valor desde la API, con fallback a la fecha actual
		valueDate := defaultValueDate
		if item.FechaActualizacion != "" {
			// La API puede retornar fechas en formato RFC3339 con zona horaria
			layouts := []string{
				"2006-01-02T15:04:05-07:00",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05.000Z",
				"2006-01-02",
			}
			for _, layout := range layouts {
				if t, parseErr := time.Parse(layout, item.FechaActualizacion); parseErr == nil {
					tVET := t.In(loc)
					valueDate = time.Date(tVET.Year(), tVET.Month(), tVET.Day(), 0, 0, 0, 0, time.UTC)
					break
				}
			}
		}

		promedio := decimal.NewFromFloat(item.Promedio)

		// Calcular compra (rate_from) con spread de 0.25% y venta (rate_to) = promedio oficial
		multiplier := decimal.NewFromFloat(0.9975)
		rateFrom := promedio.Mul(multiplier)

		// Si la API tiene compra y venta explícitas, usarlas
		if item.Compra != nil && item.Venta != nil && *item.Compra > 0 && *item.Venta > 0 {
			rateFrom = decimal.NewFromFloat(*item.Compra)
			promedio = decimal.NewFromFloat(*item.Venta)
		}

		rateAverage := rateFrom.Add(promedio).Div(decimal.NewFromInt(2))

		rate := domain.ExchangeRate{
			CurrencyCode: item.Moneda, // "USD"
			RateFrom:     rateFrom,
			RateTo:       promedio,
			RateAverage:  rateAverage,
			ValueDate:    valueDate,
		}

		rates = append(rates, rate)
		slog.Info("Tasa BCV obtenida vía dolarapi",
			"moneda", item.Moneda,
			"promedio", promedio.String(),
			"value_date", valueDate.Format("2006-01-02"),
		)
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("no se encontraron tasas oficiales del BCV en la respuesta de dolarapi")
	}

	slog.Info("Obtención de tasas BCV vía dolarapi completada", "tasas_obtenidas", len(rates))
	return rates, nil
}
