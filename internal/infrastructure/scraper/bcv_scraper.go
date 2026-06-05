package scraper

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/gocolly/colly/v2"
	"github.com/shopspring/decimal"
)

// BCVScraper implementa domain.ScraperService raspando directamente el sitio oficial del BCV (bcv.org.ve).
// El BCV publica sus tasas en la página principal dentro de bloques con IDs #dolar, #euro, etc.
type BCVScraper struct {
	url string
}

// NewBCVScraper crea e inicializa una nueva instancia de BCVScraper apuntando al sitio oficial del BCV.
func NewBCVScraper() domain.ScraperService {
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

	// Omitir la verificación de certificados TLS/SSL porque bcv.org.ve
	// a menudo usa certificados no reconocidos por los CAs estándar de producción.
	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	})

	// Mapeo de ID del div -> código de moneda ISO
	// El BCV usa: #dolar (USD), #euro (EUR), #yuan (CNY), #lira (TRY), #rublo (RUB)
	divToCode := map[string]string{
		"dolar": "USD",
		"euro":  "EUR",
		"yuan":  "CNY",
		"lira":  "TRY",
		"rublo": "RUB",
	}

	var found map[string]string
	var bcvDateText string
	var visitErr error

	maxAttempts := 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Inicializar el colector fresco para cada intento
		c := colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
			colly.AllowedDomains("www.bcv.org.ve", "bcv.org.ve"),
		)
		c.SetRequestTimeout(25 * time.Second)

		// Omitir la verificación de certificados TLS/SSL porque bcv.org.ve
		// a menudo usa certificados no reconocidos por los CAs estándar de producción.
		c.WithTransport(&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		})

		found = make(map[string]string)
		bcvDateText = ""
		var attemptErr error

		c.OnHTML("div.pull-right.dinpro.center", func(e *colly.HTMLElement) {
			bcvDateText = strings.TrimSpace(e.Text)
		})

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

		c.OnError(func(r *colly.Response, err error) {
			attemptErr = fmt.Errorf("error HTTP (Status %d) al raspar bcv.org.ve: %w", r.StatusCode, err)
			slog.Error("Fallo en la petición HTTP al BCV en intento", "intento", attempt, "status", r.StatusCode, "error", err)
		})

		c.OnRequest(func(r *colly.Request) {
			slog.Debug("Enviando petición HTTP a bcv.org.ve...", "intento", attempt, "url", r.URL.String())
		})

		// Ejecutar la visita en goroutine para respetar la cancelación del contexto
		visitChan := make(chan error, 1)
		go func() {
			visitChan <- c.Visit(s.url)
		}()

		var err error
		select {
		case <-ctx.Done():
			slog.Warn("Scraping del BCV cancelado por el contexto (timeout/cancel)")
			return nil, ctx.Err()
		case err = <-visitChan:
		}

		if err != nil || attemptErr != nil || len(found) == 0 {
			visitErr = err
			if attemptErr != nil {
				visitErr = attemptErr
			} else if len(found) == 0 {
				visitErr = fmt.Errorf("no se extrajo ninguna tasa de cambio del BCV en este intento")
			}

			slog.Warn("Fallo en intento de scraping del BCV, reintentando...",
				"intento", attempt,
				"max_intentos", maxAttempts,
				"error", visitErr,
			)

			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					backoff *= 2
				}
			}
		} else {
			// Éxito en este intento
			visitErr = nil
			break
		}
	}

	if visitErr != nil {
		return nil, fmt.Errorf("fallaron todos los %d intentos de scraping del BCV: %w", maxAttempts, visitErr)
	}

	// Fecha valor: se intenta extraer directamente de la web del BCV (div.pull-right.dinpro.center)
	// y se cae en fallback a la fecha de HOY en Venezuela (UTC-4) si algo falla.
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	if bcvDateText != "" {
		if parsedDate, err := parseSpanishDate(bcvDateText); err == nil {
			valueDate = parsedDate
			slog.Info("Fecha Valor del BCV parseada con éxito del portal", "fecha", valueDate.Format("2006-01-02"))
		} else {
			slog.Warn("No se pudo parsear la Fecha Valor del portal BCV, usando fallback (hoy en VET)", "texto", bcvDateText, "error", err)
		}
	} else {
		slog.Warn("No se encontró el texto de Fecha Valor en el HTML del BCV, usando fallback (hoy en VET)")
	}

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

// parseSpanishDate convierte un texto de fecha en español del BCV a time.Time en UTC.
// Formatos esperados del portal: "Fecha Valor: Viernes, 29 Mayo  2026", "Viernes, 29 Mayo 2026", etc.
func parseSpanishDate(text string) (time.Time, error) {
	// Remover "Fecha Valor:" si está presente
	if idx := strings.Index(text, "Fecha Valor:"); idx != -1 {
		text = text[idx+len("Fecha Valor:"):]
	}
	text = strings.TrimSpace(text)

	// Remover el día de la semana si tiene coma, ej: "Viernes, 29 Mayo 2026"
	if idx := strings.Index(text, ","); idx != -1 {
		text = text[idx+1:]
	}
	text = strings.TrimSpace(text)

	// Normalizar espacios múltiples
	fields := strings.Fields(text)
	if len(fields) < 3 {
		return time.Time{}, fmt.Errorf("formato de fecha insuficiente (se requieren al menos 3 campos): %s", text)
	}

	dayStr := fields[0]
	monthStr := strings.ToLower(fields[1])
	yearStr := fields[2]

	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error al parsear día '%s': %w", dayStr, err)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error al parsear año '%s': %w", yearStr, err)
	}

	months := map[string]time.Month{
		"enero":      time.January,
		"febrero":    time.February,
		"marzo":      time.March,
		"abril":      time.April,
		"mayo":       time.May,
		"junio":      time.June,
		"julio":      time.July,
		"agosto":     time.August,
		"septiembre": time.September,
		"setiembre":  time.September,
		"octubre":    time.October,
		"noviembre":  time.November,
		"diciembre":  time.December,
	}

	month, ok := months[monthStr]
	if !ok {
		return time.Time{}, fmt.Errorf("mes en español desconocido: %s", monthStr)
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
