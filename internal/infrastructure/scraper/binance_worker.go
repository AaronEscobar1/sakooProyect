package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/currency"
	"github.com/aaron/sakoo-backend/ent/exchangerate"
	"github.com/aaron/sakoo-backend/internal/infrastructure/notification"
	"github.com/shopspring/decimal"
)

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

type BinanceSearchRequest struct {
	Fiat       string   `json:"fiat"`
	Page       int      `json:"page"`
	Rows       int      `json:"rows"`
	TradeType  string   `json:"tradeType"`
	Asset      string   `json:"asset"`
	PayTypes   []string `json:"payTypes"`
	Classifies []string `json:"classifies"`
}

func RunBinanceWorker(ctx context.Context, client *ent.Client, targetAsset string) error {
	loc := time.FixedZone("America/Caracas", -4*60*60)
	nowVET := time.Now().In(loc)
	valueDate := time.Date(nowVET.Year(), nowVET.Month(), nowVET.Day(), 0, 0, 0, 0, time.UTC)

	slog.Info("Iniciando Binance P2P Worker con Ent...",
		"asset", targetAsset,
		"hora_venezuela", nowVET.Format("2006-01-02 15:04:05"),
		"fecha_valor", valueDate.Format("2006-01-02"),
	)

	var assetName string
	switch targetAsset {
	case "USDT":
		assetName = "Tether USDT"
	case "USDC":
		assetName = "USD Coin"
	default:
		assetName = targetAsset + " Cripto"
	}

	// Obtener/crear divisa en Ent
	curr, err := client.Currency.Query().Where(currency.CodeEQ(targetAsset)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			curr, err = client.Currency.Create().SetCode(targetAsset).SetName(assetName).Save(ctx)
			if err != nil {
				return fmt.Errorf("error al crear divisa %s: %w", targetAsset, err)
			}
		} else {
			return fmt.Errorf("error al buscar divisa %s: %w", targetAsset, err)
		}
	}
	currencyID := int64(curr.ID)

	purchaseRate, err := fetchBinanceP2PAverage(ctx, targetAsset, "BUY")
	if err != nil {
		return fmt.Errorf("error al obtener tasa de compra para %s: %w", targetAsset, err)
	}

	saleRate, err := fetchBinanceP2PAverage(ctx, targetAsset, "SELL")
	if err != nil {
		return fmt.Errorf("error al obtener tasa de venta para %s: %w", targetAsset, err)
	}

	avgGlobal := purchaseRate.Add(saleRate).Div(decimal.NewFromInt(2))

	purchaseFloat, _ := purchaseRate.Float64()
	saleFloat, _ := saleRate.Float64()
	avgGlobalFloat, _ := avgGlobal.Float64()

	existing, err := client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(currencyID),
			exchangerate.ValueDateEQ(valueDate),
		).
		Only(ctx)

	sendPush := false
	if err != nil && ent.IsNotFound(err) {
		sendPush = true
		_, err = client.ExchangeRate.Create().
			SetCurrencyID(currencyID).
			SetRateFrom(purchaseFloat).
			SetRateTo(saleFloat).
			SetRateAverage(avgGlobalFloat).
			SetValueDate(valueDate).
			Save(ctx)
	} else if err == nil {
		_, err = client.ExchangeRate.UpdateOne(existing).
			SetRateFrom(purchaseFloat).
			SetRateTo(saleFloat).
			SetRateAverage(avgGlobalFloat).
			SetUpdatedAt(time.Now()).
			Save(ctx)
	}

	if err != nil {
		return fmt.Errorf("error al actualizar tabla de tasas en Ent: %w", err)
	}

	slog.Info("Tabla principal de tasas actualizada con éxito", "asset", targetAsset)

	if sendPush {
		rateStr := saleRate.Truncate(2).StringFixed(2)
		slog.Info("Primer ciclo de Binance P2P del día. Enviando notificación push...", "asset", targetAsset, "rate", rateStr)

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

	return nil
}

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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := httpClient.Do(req)
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
			continue
		}
		sum = sum.Add(priceDec)
		count++
	}

	if count == 0 {
		return decimal.Zero, fmt.Errorf("ninguno de los precios de los anuncios de Binance P2P pudo ser procesado como decimal")
	}

	return sum.Div(decimal.NewFromInt(int64(count))), nil
}
