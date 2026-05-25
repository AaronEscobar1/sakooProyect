package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
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

// BCVScraper implementa ScraperService para extraer tasas de cambio desde el Banco Central de Venezuela.
type BCVScraper struct {
	url string
}

var spanishMonths = map[string]time.Month{
	"enero":      time.January,
	"febrero":    time.February,
	"marzo":      time.March,
	"abril":      time.April,
	"mayo":       time.May,
	"junio":      time.June,
	"julio":      time.July,
	"agosto":     time.August,
	"septiembre": time.September,
	"octubre":    time.October,
	"noviembre":  time.November,
	"diciembre":  time.December,
}

func parseBCVDate(text string) (time.Time, error) {
	// Eliminar el prefijo "Fecha Valor:" si existe
	cleaned := strings.ReplaceAll(text, "Fecha Valor:", "")
	cleaned = strings.TrimSpace(cleaned)

	// Si hay una coma, tomamos lo que está a la derecha (ej: "Lunes, 25 Mayo 2026")
	if idx := strings.Index(cleaned, ","); idx != -1 {
		cleaned = cleaned[idx+1:]
	}
	cleaned = strings.TrimSpace(cleaned)

	// Reemplazar múltiples espacios consecutivos por uno solo
	fields := strings.Fields(cleaned)
	if len(fields) < 3 {
		return time.Time{}, fmt.Errorf("formato de fecha de valor de BCV inválido: %s", text)
	}

	dayStr := fields[0]
	monthStr := strings.ToLower(fields[1])
	yearStr := fields[2]

	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error al convertir día de BCV: %w", err)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("error al convertir año de BCV: %w", err)
	}

	month, exists := spanishMonths[monthStr]
	if !exists {
		return time.Time{}, fmt.Errorf("mes en español desconocido de BCV: %s", monthStr)
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

// NewBCVScraper crea e inicializa una nueva instancia de BCVScraper.
func NewBCVScraper() ScraperService {
	return &BCVScraper{
		url: "https://www.bcv.org.ve/",
	}
}

// ScrapeRates extrae de manera resiliente las tasas oficiales de USD, EUR, CNY, TRY y RUB desde el BCV.
func (s *BCVScraper) ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	slog.Info("Iniciando raspado de tasas del Banco Central de Venezuela (BCV)...", "url", s.url)

	// 1. Configurar coleccionador principal con User-Agent de navegador real moderno
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowedDomains("www.bcv.org.ve", "bcv.org.ve"),
	)

	// 2. Resiliencia: Configurar un timeout de al menos 15 segundos (usaremos 20 segundos)
	c.SetRequestTimeout(20 * time.Second)

	// 3. Resiliencia: Establecer colly.LimitRule con retraso aleatorio de 1 a 3 segundos para evitar ser bloqueados
	err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*bcv.org.ve*",
		Parallelism: 1,
		Delay:       1 * time.Second,
		RandomDelay: 2 * time.Second,
	})
	if err != nil {
		slog.Warn("No se pudo aplicar la regla de límites en Colly. Continuando de forma estándar...", "error", err)
	}

	// Mapa temporal para recolectar los valores raspados
	scrapedMap := make(map[string]string)
	var dateText string

	// 4. Parsing: Selectores CSS exactos y estables de las divisas en el portal del BCV
	selectors := map[string]string{
		"USD": "#dolar strong",
		"EUR": "#euro strong",
		"CNY": "#yuan strong",
		"TRY": "#lira strong",
		"RUB": "#rublo strong",
	}

	// Registrar los callbacks de OnHTML para cada selector de moneda
	for currency, selector := range selectors {
		code := currency // Captura local para la goroutine/closure
		sel := selector  // Captura local

		c.OnHTML(sel, func(e *colly.HTMLElement) {
			valText := strings.TrimSpace(e.Text)
			slog.Debug("Tasa cruda encontrada en el DOM de la web", "moneda", code, "selector", sel, "valor", valText)
			if valText != "" {
				scrapedMap[code] = valText
			}
		})
	}

	// Capturar la fecha valor de la página del BCV de manera robusta
	c.OnHTML("div.pull-right.dinpro", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if strings.Contains(text, "Fecha Valor:") {
			dateText = text
			slog.Debug("Fecha Valor encontrada en div.pull-right.dinpro", "texto", dateText)
		}
	})

	c.OnHTML("div", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if dateText == "" && strings.Contains(text, "Fecha Valor:") && len(text) < 100 {
			dateText = text
			slog.Debug("Fecha Valor encontrada vía fallback en div", "texto", dateText)
		}
	})

	// Manejo robusto de errores de conexión y red
	var visitErr error
	c.OnError(func(r *colly.Response, err error) {
		visitErr = fmt.Errorf("error de conexión HTTP (Status %d) en BCV: %w", r.StatusCode, err)
		slog.Error("Fallo en la petición HTTP del scraper", "status", r.StatusCode, "url", r.Request.URL.String(), "error", err)
	})

	c.OnRequest(func(r *colly.Request) {
		slog.Debug("Enviando petición HTTP...", "url", r.URL.String())
	})

	// Ejecutar la visita en una goroutine para poder soportar cancelación del contexto
	visitChan := make(chan error, 1)
	go func() {
		visitChan <- c.Visit(s.url)
	}()

	// Esperar completitud de raspado o cancelación del contexto (resiliencia)
	select {
	case <-ctx.Done():
		slog.Warn("La operación de scraping fue cancelada por el contexto del llamador (timeout/cancel)")
		return nil, ctx.Err()
	case err := <-visitChan:
		if err != nil {
			return nil, fmt.Errorf("error de visita en la URL del scraper: %w", err)
		}
	}

	if visitErr != nil {
		return nil, visitErr
	}

	// 5. Parseo y limpieza de datos usando shopspring/decimal
	var rates []domain.ExchangeRate
	// Resolver la fecha de valor
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	if dateText != "" {
		parsedDate, err := parseBCVDate(dateText)
		if err != nil {
			slog.Warn("⚠️ No se pudo parsear la 'Fecha Valor' de la página del BCV. Usando fecha del sistema como fallback.", "texto", dateText, "error", err)
		} else {
			valueDate = parsedDate
			slog.Info("📅 Fecha Valor de la tasa de cambio extraída exitosamente de la página del BCV", "fecha", valueDate.Format("2006-01-02"))
		}
	} else {
		slog.Warn("⚠️ No se encontró el elemento de 'Fecha Valor' en la página del BCV. Usando fecha del sistema como fallback.")
	}

	for code, rawValue := range scrapedMap {
		// Limpieza del string: quitar espacios, cambiar comas por puntos (formato decimal Go)
		cleanValue := strings.TrimSpace(rawValue)
		cleanValue = strings.ReplaceAll(cleanValue, ",", ".")

		parsedDecimal, err := decimal.NewFromString(cleanValue)
		if err != nil {
			slog.Error("Fallo al convertir la tasa de texto a decimal.Decimal", 
				"moneda", code, 
				"texto_crudo", rawValue, 
				"texto_limpio", cleanValue, 
				"error", err,
			)
			continue // Continuar procesando las demás monedas
		}

		// Calcular rate_from (rate_to * 0.9975 para spread de compra de 0.25%)
		multiplier := decimal.NewFromFloat(0.9975)
		rateFrom := parsedDecimal.Mul(multiplier)

		// Calcular rate_average como promedio simple: (rate_from + rate_to) / 2
		two := decimal.NewFromInt(2)
		rateAverage := rateFrom.Add(parsedDecimal).Div(two)

		// Construir la entidad con la equivalencia de 1 Unidad Extranjera a X Unidades Locales (VES)
		rate := domain.ExchangeRate{
			CurrencyCode: code,
			RateFrom:     rateFrom,      // Tasa de compra calculada con spread
			RateTo:       parsedDecimal, // Tasa de venta oficial del BCV
			RateAverage:  rateAverage,   // Promedio ponderado
			ValueDate:    valueDate,
		}

		rates = append(rates, rate)
		slog.Debug("Tasa parseada correctamente", "moneda", code, "valor", parsedDecimal.String())
	}

	slog.Info("Proceso de scraping del BCV completado exitosamente", "tasas_extraidas", len(rates))
	return rates, nil
}
