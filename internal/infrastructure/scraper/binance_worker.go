package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aaron/sakoo-backend/internal/infrastructure/notification"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// BinanceResponse representa la respuesta estructurada de la API de Binance P2P.
type BinanceResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		Adv struct {
			Price string `json:"price"`
		} `json:"adv"`
	} `json:"data"`
	Success bool `json:"success"`
}

// BinanceSearchRequest representa el payload JSON para buscar ofertas de P2P en Binance.
type BinanceSearchRequest struct {
	Fiat       string   `json:"fiat"`
	Page       int      `json:"page"`
	Rows       int      `json:"rows"`
	TradeType  string   `json:"tradeType"`
	Asset      string   `json:"asset"`
	PayTypes   []string `json:"payTypes"`
	Classifies []string `json:"classifies"`
}

// RunBinanceWorker ejecuta el ciclo de trabajo recurrente para una criptodivisa específica en Binance P2P.
// Crea la tabla e índice de logs al vuelo, asegura la moneda en el catálogo, realiza llamadas HTTP
// para tasas de compra/venta, guarda muestras, calcula el promedio diario y actualiza la tabla de tasas.
func RunBinanceWorker(ctx context.Context, db *pgxpool.Pool, targetAsset string) error {
	// Calcular fechas y zonas horarias basadas en la hora de Venezuela (UTC-4)
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	startOfDayVET := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	slog.Info("Iniciando Binance P2P Worker...", 
		"asset", targetAsset, 
		"hora_venezuela", nowVET.Format("2006-01-02 15:04:05"), 
		"fecha_valor", valueDate.Format("2006-01-02"),
	)

	// Paso A: Crear tabla e índices si no existen (Auto-curación)
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS market.exchange_rate_samples (
			id BIGSERIAL PRIMARY KEY,
			currency_id BIGINT NOT NULL REFERENCES catalogs.currency(id) ON DELETE CASCADE,
			rate_purchase NUMERIC(10, 2) NOT NULL,
			rate_sale NUMERIC(10, 2) NOT NULL,
			sampled_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	if _, err := db.Exec(ctx, createTableQuery); err != nil {
		return fmt.Errorf("error al asegurar la existencia de market.exchange_rate_samples: %w", err)
	}

	// Paso A.1: Asegurar que el catálogo soporte códigos de moneda de hasta 4 caracteres (ej: USDT, USDC)
	alterCodeTypeQuery := `
		ALTER TABLE catalogs.currency ALTER COLUMN code TYPE VARCHAR(4);
	`
	if _, err := db.Exec(ctx, alterCodeTypeQuery); err != nil {
		slog.Warn("No se pudo alterar la longitud de columna code en catalogs.currency, asumiendo ya compatible", "error", err)
	}

	createIndexQuery := `
		CREATE INDEX IF NOT EXISTS idx_exchange_rate_samples_currency_sampled_at 
		ON market.exchange_rate_samples (currency_id, sampled_at);
	`
	if _, err := db.Exec(ctx, createIndexQuery); err != nil {
		slog.Warn("No se pudo crear el índice optimizado para muestras", "error", err)
	}

	// Paso 0: Garantizar y Obtener el ID de la Moneda
	var assetName string
	switch targetAsset {
	case "USDT":
		assetName = "Tether USDT"
	case "USDC":
		assetName = "USD Coin"
	default:
		assetName = targetAsset + " Cripto"
	}

	var currencyID int64
	upsertCurrencyQuery := `
		INSERT INTO catalogs.currency (code, name)
		VALUES ($1, $2)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id;
	`
	err := db.QueryRow(ctx, upsertCurrencyQuery, targetAsset, assetName).Scan(&currencyID)
	if err != nil {
		return fmt.Errorf("error al obtener/crear id de divisa %s: %w", targetAsset, err)
	}
	slog.Info("ID de divisa obtenido con éxito en catálogo", "asset", targetAsset, "currency_id", currencyID)

	// Paso 1: Scraping API Binance P2P (COMPRA)
	purchaseRate, err := fetchBinanceP2PAverage(ctx, targetAsset, "BUY")
	if err != nil {
		return fmt.Errorf("error al obtener tasa de compra para %s: %w", targetAsset, err)
	}
	slog.Info("Tasa de COMPRA promediada obtenida de Binance P2P", "asset", targetAsset, "rate", purchaseRate.String())

	// Paso 1: Scraping API Binance P2P (VENTA)
	saleRate, err := fetchBinanceP2PAverage(ctx, targetAsset, "SELL")
	if err != nil {
		return fmt.Errorf("error al obtener tasa de venta para %s: %w", targetAsset, err)
	}
	slog.Info("Tasa de VENTA promediada obtenida de Binance P2P", "asset", targetAsset, "rate", saleRate.String())

	// Paso 2: Registrar Muestra (Insert)
	insertSampleQuery := `
		INSERT INTO market.exchange_rate_samples (currency_id, rate_purchase, rate_sale, sampled_at)
		VALUES ($1, $2, $3, NOW());
	`
	_, err = db.Exec(ctx, insertSampleQuery, currencyID, purchaseRate, saleRate)
	if err != nil {
		return fmt.Errorf("error al insertar muestra en market.exchange_rate_samples: %w", err)
	}
	slog.Info("Muestra de tasa de cambio registrada en la base de datos", "currency_id", currencyID, "purchase", purchaseRate.String(), "sale", saleRate.String())

	// Paso 3: Calcular Promedio Acumulado del Día (Select)
	var avgPurchase, avgSale, avgGlobal decimal.Decimal
	aggregateQuery := `
		SELECT 
			COALESCE(AVG(rate_purchase), 0) as avg_purchase,
			COALESCE(AVG(rate_sale), 0) as avg_sale,
			COALESCE((AVG(rate_purchase) + AVG(rate_sale)) / 2, 0) as avg_global
		FROM market.exchange_rate_samples
		WHERE currency_id = $1 AND sampled_at >= $2;
	`
	err = db.QueryRow(ctx, aggregateQuery, currencyID, startOfDayVET).Scan(&avgPurchase, &avgSale, &avgGlobal)
	if err != nil {
		return fmt.Errorf("error al calcular promedio acumulado diario: %w", err)
	}
	slog.Info("Promedios acumulados diarios consolidados", 
		"asset", targetAsset, 
		"avg_purchase", avgPurchase.String(), 
		"avg_sale", avgSale.String(), 
		"avg_global", avgGlobal.String(),
	)

	// Paso 4: Actualizar Tabla Principal (Upsert)
	// Verificar si la tasa de Binance ha cambiado respecto a la última registrada en la base de datos antes de actualizar
	var prevRateTo decimal.Decimal
	var prevValueDate time.Time
	hasPrev := false
	checkQuery := `
		SELECT rate_to, value_date 
		FROM market.exchange_rates 
		WHERE currency_id = $1 
		ORDER BY value_date DESC, id DESC 
		LIMIT 1;
	`
	errCheck := db.QueryRow(ctx, checkQuery, currencyID).Scan(&prevRateTo, &prevValueDate)
	if errCheck == nil {
		hasPrev = true
	}

	// Las notificaciones de Binance se envían una sola vez al día (cuando cambia la fecha valor o no hay registro previo).
	// Esto evita spam por fluctuaciones horarias constantes en el mercado P2P.
	sendPush := false
	if !hasPrev {
		sendPush = true
	} else if prevValueDate.UTC().Format("2006-01-02") != valueDate.UTC().Format("2006-01-02") {
		sendPush = true
	}

	upsertMainQuery := `
		INSERT INTO market.exchange_rates (currency_id, rate_from, rate_to, rate_average, value_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (currency_id, value_date)
		DO UPDATE SET 
			rate_from = EXCLUDED.rate_from,
			rate_to = EXCLUDED.rate_to,
			rate_average = EXCLUDED.rate_average,
			updated_at = NOW();
	`
	_, err = db.Exec(ctx, upsertMainQuery, currencyID, avgPurchase, avgSale, avgGlobal, valueDate)
	if err != nil {
		return fmt.Errorf("error al realizar upsert en la tabla market.exchange_rates: %w", err)
	}
	slog.Info("Tabla principal de tasas de cambio actualizada con éxito", "asset", targetAsset)

	// Si es el primer ciclo de Binance P2P del día, disparar la notificación push al Topic de forma asíncrona
	if sendPush {
		rateStr := avgSale.Truncate(2).StringFixed(2)
		slog.Info("Primer ciclo de Binance P2P del día. Enviando notificación push diaria...", "asset", targetAsset, "rate", rateStr)
		
		// Instanciar el servicio de notificaciones al vuelo
		pushSrv := notification.NewPushNotificationService()
		title := fmt.Sprintf("¡La tasa de %s (Binance P2P) ha cambiado! 🚀", targetAsset)
		body := fmt.Sprintf("La nueva tasa de Binance P2P es de %s Bs.", rateStr)
		fcmData := map[string]string{
			"type":          "rate_update",
			"source":        "BINANCE",
			"currency_code": targetAsset,
			"rate":          rateStr,
		}

		go func() {
			bgCtx := context.Background()
			_ = pushSrv.SendTopicPush(bgCtx, "exchange_rates", title, body, fcmData)
		}()
	}

	// Paso 5: Purgar de forma segura muestras de días anteriores para mantener base de datos limpia
	pruneQuery := `
		DELETE FROM market.exchange_rate_samples
		WHERE currency_id = $1 AND sampled_at < $2;
	`
	pruneRes, err := db.Exec(ctx, pruneQuery, currencyID, startOfDayVET)
	if err != nil {
		slog.Error("Fallo al purgar muestras antiguas en Binance P2P Worker", "asset", targetAsset, "error", err)
	} else {
		slog.Info("Muestras de días anteriores purgadas correctamente", "asset", targetAsset, "filas_afectadas", pruneRes.RowsAffected())
	}

	return nil
}

// fetchBinanceP2PAverage consulta la API oficial P2P de Binance, extrae las primeras 5 ofertas válidas y calcula el promedio matemático simple.
func fetchBinanceP2PAverage(ctx context.Context, asset string, tradeType string) (decimal.Decimal, error) {
	apiURL := "https://p2p.binance.com/bapi/c2c/v2/friendly/c2c/adv/search"

	reqPayload := BinanceSearchRequest{
		Fiat:       "VES",
		Page:       1,
		Rows:       10,
		TradeType:  tradeType,
		Asset:      asset,
		PayTypes:   []string{},
		Classifies: []string{"mass", "profession"},
	}

	jsonPayload, err := json.Marshal(reqPayload)
	if err != nil {
		return decimal.Zero, fmt.Errorf("error al serializar payload de Binance: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return decimal.Zero, fmt.Errorf("error al crear petición HTTP a Binance: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Usar un User-Agent realista de navegador moderno
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("error de red al consultar Binance P2P: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("código de estado HTTP no exitoso de Binance P2P: %d", resp.StatusCode)
	}

	var binanceResp BinanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&binanceResp); err != nil {
		return decimal.Zero, fmt.Errorf("error al decodificar respuesta JSON de Binance P2P: %w", err)
	}

	if len(binanceResp.Data) == 0 {
		return decimal.Zero, fmt.Errorf("no se encontraron anuncios activos para %s en Binance P2P", asset)
	}

	// Iterar las primeras 5 ofertas (o menos si hay menos ofertas disponibles)
	limit := 5
	if len(binanceResp.Data) < limit {
		limit = len(binanceResp.Data)
	}

	var sum decimal.Decimal
	var count int

	for i := 0; i < limit; i++ {
		priceStr := binanceResp.Data[i].Adv.Price
		priceDec, err := decimal.NewFromString(priceStr)
		if err != nil {
			slog.Warn("Precio no numérico en Binance P2P, omitiendo anuncio", "price", priceStr, "error", err)
			continue
		}
		sum = sum.Add(priceDec)
		count++
	}

	if count == 0 {
		return decimal.Zero, fmt.Errorf("ninguno de los precios de los anuncios de Binance P2P pudo ser procesado como decimal")
	}

	avg := sum.Div(decimal.NewFromInt(int64(count)))
	return avg, nil
}
