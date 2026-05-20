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

// BCVScraper implementa ScraperService para extraer tasas de cambio desde el Banco Central de Venezuela.
type BCVScraper struct {
	url string
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
	now := time.Now().UTC()
	// La columna value_date de base de datos es de tipo DATE. La truncamos a las 00:00:00 UTC
	valueDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

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

		// Construir la entidad con la equivalencia de 1 Unidad Extranjera a X Unidades Locales (VES)
		rate := domain.ExchangeRate{
			CurrencyCode: code,
			RateFrom:     decimal.NewFromInt(1), // 1.0 de la moneda origen (ej: 1 USD)
			RateTo:       parsedDecimal,         // Valor de destino (ej: 36.54 VES)
			RateAverage:  parsedDecimal,         // Tasa promedio oficial publicada
			ValueDate:    valueDate,
		}

		rates = append(rates, rate)
		slog.Debug("Tasa parseada correctamente", "moneda", code, "valor", parsedDecimal.String())
	}

	slog.Info("Proceso de scraping del BCV completado exitosamente", "tasas_extraidas", len(rates))
	return rates, nil
}
