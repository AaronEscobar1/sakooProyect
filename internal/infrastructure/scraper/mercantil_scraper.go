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

// MercantilScraper implementa ScraperService para extraer la tasa "Mesa de Cambio" del Banco Mercantil.
type MercantilScraper struct {
	url string
}

// NewMercantilScraper crea e inicializa una nueva instancia de MercantilScraper.
func NewMercantilScraper() domain.ScraperService {
	return &MercantilScraper{
		url: "https://www.mercantilbanco.com/informacion/tasas,-tarifas-y-comisiones/tasa-mesa-de-cambio",
	}
}

// ScrapeRates extrae de manera resiliente la tasa oficial del dólar de intervención (Mesa de Cambio) del Mercantil Banco.
func (s *MercantilScraper) ScrapeRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	slog.Info("Iniciando raspado de tasas del Mercantil Banco...", "url", s.url)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(20 * time.Second)

	var dateStr, compraStr, ventaStr string
	var visitErr error

	maxAttempts := 3
	backoff := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Inicializar colector fresco en cada intento
		c := colly.NewCollector(
			colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		)
		c.SetRequestTimeout(20 * time.Second)

		var tableIndex int
		dateStr = ""
		compraStr = ""
		ventaStr = ""
		var attemptErr error

		// Escuchar sobre las tablas de la página
		c.OnHTML("table", func(e *colly.HTMLElement) {
			tableIndex++
			// Solo nos interesa la primera tabla que contiene la cotización del dólar de Mercantil
			if tableIndex != 1 {
				return
			}

			e.ForEach("tbody tr", func(_ int, el *colly.HTMLElement) {
				el.ForEach("td", func(i int, cell *colly.HTMLElement) {
					text := strings.TrimSpace(cell.Text)
					switch i {
					case 0:
						dateStr = text
					case 1:
						compraStr = text
					case 2:
						ventaStr = text
					}
				})
			})
		})

		c.OnError(func(r *colly.Response, err error) {
			attemptErr = fmt.Errorf("error de conexión HTTP (Status %d) en Mercantil: %w", r.StatusCode, err)
			slog.Error("Fallo en la petición HTTP del scraper Mercantil en intento", "intento", attempt, "status", r.StatusCode, "error", err)
		})

		c.OnRequest(func(r *colly.Request) {
			slog.Debug("Enviando petición HTTP a Mercantil...", "intento", attempt, "url", r.URL.String())
		})

		// Ejecutar la visita en una goroutine para soportar cancelación del contexto
		visitChan := make(chan error, 1)
		go func() {
			visitChan <- c.Visit(s.url)
		}()

		var err error
		select {
		case <-ctx.Done():
			slog.Warn("La operación de scraping Mercantil fue cancelada por el contexto (timeout/cancel)")
			return nil, ctx.Err()
		case err = <-visitChan:
		}

		if err != nil || attemptErr != nil || ventaStr == "" || compraStr == "" {
			visitErr = err
			if attemptErr != nil {
				visitErr = attemptErr
			} else if ventaStr == "" || compraStr == "" {
				visitErr = fmt.Errorf("no se encontró el valor de la tasa de compra o venta en el HTML de Mercantil")
			}

			slog.Warn("Fallo en intento de scraping de Mercantil, reintentando...",
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
			// Éxito
			visitErr = nil
			break
		}
	}

	if visitErr != nil {
		return nil, fmt.Errorf("fallaron todos los %d intentos de scraping de Mercantil: %w", maxAttempts, visitErr)
	}

	// 1. Parsear los valores de las tasas
	cleanVenta := strings.ReplaceAll(ventaStr, ",", ".")
	parsedVenta, err := decimal.NewFromString(cleanVenta)
	if err != nil {
		slog.Error("Fallo al convertir la tasa de venta de Mercantil", "error", err)
		return nil, fmt.Errorf("error al parsear tasa de venta: %w", err)
	}

	cleanCompra := strings.ReplaceAll(compraStr, ",", ".")
	parsedCompra, err := decimal.NewFromString(cleanCompra)
	if err != nil {
		slog.Error("Fallo al convertir la tasa de compra de Mercantil", "error", err)
		return nil, fmt.Errorf("error al parsear tasa de compra: %w", err)
	}

	parsedAverage := parsedCompra.Add(parsedVenta).Div(decimal.NewFromInt(2))

	// 2. Parsear la fecha valor, con fallback a la fecha del sistema truncada
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	if dateStr != "" {
		parsedDate, err := time.Parse("02/01/2006", dateStr)
		if err == nil {
			valueDate = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, time.UTC)
		} else {
			slog.Warn("No se pudo parsear la fecha valor extraída, utilizando fecha actual del sistema", "fecha_extraida", dateStr, "error", err)
		}
	}

	// 3. Retornar el slice con un único elemento (UDI)
	rate := domain.ExchangeRate{
		CurrencyCode: "UDI",
		RateFrom:     parsedCompra, // TC Compra
		RateTo:       parsedVenta,  // TC Venta
		RateAverage:  parsedAverage, // Promedio de ambas
		ValueDate:    valueDate,
	}

	rates := []domain.ExchangeRate{rate}
	slog.Info("Proceso de scraping de Mercantil completado", "tasas_extraidas", len(rates), "compra", parsedCompra.String(), "venta", parsedVenta.String())

	return rates, nil
}
