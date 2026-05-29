package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/gocolly/colly/v2"
	"github.com/shopspring/decimal"
)

// ScraperService define el contrato para el servicio de raspado de tasas de cambio.
type ScraperService interface {
	ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error)
}

// BCVScraper implementa ScraperService raspando directamente el sitio oficial del BCV (bcv.org.ve).
// El BCV publica sus tasas en la página principal dentro de bloques con IDs #dolar, #euro, etc.
type BCVScraper struct {
	url string
}

// NewBCVScraper crea e inicializa una nueva instancia de BCVScraper apuntando al sitio oficial del BCV.
func NewBCVScraper() ScraperService {
	return &BCVScraper{
		url: "https://www.bcv.org.ve/",
	}
}

// bcvRateResult acumula los datos de una moneda durante el scraping.
type bcvRateResult struct {
	currencyCode string
	rateStr      string
}

// ScrapeRates raspa las tasas oficiales del BCV directamente desde bcv.org.ve.
// Extrae USD y EUR (los bloques #dolar y #euro de la homepage).
// La value_date es siempre HOY en Venezuela (UTC-4), independientemente de lo que
// publique el BCV, para garantizar que el Upsert siempre cree un registro nuevo por día.
func (s *BCVScraper) ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	slog.Info("Iniciando raspado de tasas del BCV directamente desde bcv.org.ve...", "url", s.url)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
		colly.AllowedDomains("www.bcv.org.ve", "bcv.org.ve"),
	)
	c.SetRequestTimeout(25 * time.Second)

	// Mapeo de ID del div -> código de moneda ISO
	// El BCV usa: #dolar (USD), #euro (EUR), #yuan (CNY), #lira (TRY), #rublo (RUB)
	divToCode := map[string]string{
		"dolar": "USD",
		"euro":  "EUR",
	}

	// Recolector de resultados durante el scraping
	found := make(map[string]string) // currencyCode -> rateStr

	// El BCV publica cada divisa en un bloque como:
	//   <div id="dolar" class="...">
	//     <div class="centrado">
	//       <strong>36,4875</strong>
	//     </div>
	//   </div>
	for divID, code := range divToCode {
		capturedCode := code // capturar para el closure
		selector := fmt.Sprintf("div#%s div.centrado strong", divID)
		c.OnHTML(selector, func(e *colly.HTMLElement) {
			raw := strings.TrimSpace(e.Text)
			if raw == "" {
				return
			}
			// El BCV usa coma como separador decimal (ej: "36,4875") → convertir a punto
			normalized := strings.ReplaceAll(raw, ".", "")  // quitar separadores de miles
			normalized = strings.ReplaceAll(normalized, ",", ".") // coma decimal → punto
			slog.Debug("Tasa BCV raspada del HTML", "moneda", capturedCode, "raw", raw, "normalizado", normalized)
			found[capturedCode] = normalized
		})
	}

	var visitErr error
	c.OnError(func(r *colly.Response, err error) {
		visitErr = fmt.Errorf("error HTTP (Status %d) al raspar bcv.org.ve: %w", r.StatusCode, err)
		slog.Error("Fallo en la petición HTTP al BCV", "status", r.StatusCode, "url", r.Request.URL.String(), "error", err)
	})

	c.OnRequest(func(r *colly.Request) {
		slog.Debug("Enviando petición HTTP a bcv.org.ve...", "url", r.URL.String())
	})

	// Ejecutar la visita en goroutine para respetar la cancelación del contexto
	visitChan := make(chan error, 1)
	go func() {
		visitChan <- c.Visit(s.url)
	}()

	select {
	case <-ctx.Done():
		slog.Warn("Scraping del BCV cancelado por el contexto (timeout/cancel)")
		return nil, ctx.Err()
	case err := <-visitChan:
		if err != nil {
			return nil, fmt.Errorf("error al visitar bcv.org.ve: %w", err)
		}
	}

	if visitErr != nil {
		return nil, visitErr
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no se encontraron tasas del BCV en bcv.org.ve (el sitio puede haber cambiado su estructura HTML)")
	}

	// Fecha valor: siempre HOY en Venezuela (UTC-4), truncado al día.
	// NO usamos la fecha que publica el BCV porque puede llevar días sin actualizarse
	// (ej: publicó el 26-may y el cartel sigue igual el 27 y 28). Si usáramos esa fecha,
	// el Upsert pisaría el registro viejo en lugar de crear uno nuevo para hoy.
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	var rates []domain.ExchangeRate

	for currencyCode, rateStr := range found {
		promedio, err := decimal.NewFromString(rateStr)
		if err != nil || promedio.IsZero() || promedio.IsNegative() {
			slog.Warn("Tasa BCV con valor inválido ignorada", "moneda", currencyCode, "valor", rateStr, "error", err)
			continue
		}

		// Aplicar spread de 0.25% para obtener el precio de compra (rate_from)
		// y usar el promedio publicado como precio de venta (rate_to)
		multiplier := decimal.NewFromFloat(0.9975)
		rateFrom := promedio.Mul(multiplier)
		rateAverage := rateFrom.Add(promedio).Div(decimal.NewFromInt(2))

		rate := domain.ExchangeRate{
			CurrencyCode: currencyCode,
			RateFrom:     rateFrom,
			RateTo:       promedio,
			RateAverage:  rateAverage,
			ValueDate:    valueDate,
		}

		rates = append(rates, rate)
		slog.Info("Tasa BCV obtenida directamente de bcv.org.ve",
			"moneda", currencyCode,
			"promedio", promedio.String(),
			"value_date", valueDate.Format("2006-01-02"),
		)
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("no se pudieron parsear las tasas del BCV (valores vacíos o inválidos)")
	}

	slog.Info("Raspado de tasas BCV desde bcv.org.ve completado", "tasas_obtenidas", len(rates))
	return rates, nil
}
