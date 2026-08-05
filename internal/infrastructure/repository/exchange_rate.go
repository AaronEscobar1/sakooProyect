package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aaron/sakoo-backend/ent"
	"github.com/aaron/sakoo-backend/ent/currency"
	"github.com/aaron/sakoo-backend/ent/exchangerate"
	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/shopspring/decimal"
)

type exchangeRateRepository struct {
	client *ent.Client
}

// NewExchangeRateRepository crea una nueva instancia del repositorio de tasas de cambio usando Ent.
func NewExchangeRateRepository(client *ent.Client) domain.ExchangeRateRepository {
	return &exchangeRateRepository{
		client: client,
	}
}

func (r *exchangeRateRepository) Upsert(ctx context.Context, rate *domain.ExchangeRate) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rateFromFloat, _ := rate.RateFrom.Float64()
	rateToFloat, _ := rate.RateTo.Float64()
	rateAvgFloat, _ := rate.RateAverage.Float64()

	existing, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(rate.CurrencyID),
			exchangerate.ValueDateEQ(rate.ValueDate),
		).
		Only(dbCtx)

	if err != nil && !ent.IsNotFound(err) {
		slog.Error("Fallo al buscar tasa de cambio previa en Ent", "error", err)
		return fmt.Errorf("error al persistir tasa de cambio (upsert): %w", err)
	}

	if existing != nil {
		updated, err := r.client.ExchangeRate.UpdateOne(existing).
			SetRateFrom(rateFromFloat).
			SetRateTo(rateToFloat).
			SetRateAverage(rateAvgFloat).
			SetUpdatedAt(time.Now()).
			Save(dbCtx)
		if err != nil {
			slog.Error("Fallo al actualizar tasa en Ent", "error", err)
			return fmt.Errorf("error al persistir tasa de cambio (upsert): %w", err)
		}
		rate.ID = int64(updated.ID)
	} else {
		created, err := r.client.ExchangeRate.Create().
			SetCurrencyID(rate.CurrencyID).
			SetRateFrom(rateFromFloat).
			SetRateTo(rateToFloat).
			SetRateAverage(rateAvgFloat).
			SetValueDate(rate.ValueDate).
			Save(dbCtx)
		if err != nil {
			slog.Error("Fallo al crear tasa en Ent", "error", err)
			return fmt.Errorf("error al persistir tasa de cambio (upsert): %w", err)
		}
		rate.ID = int64(created.ID)
	}

	slog.Info("Tasa de cambio persistida correctamente en Ent",
		"id", rate.ID,
		"currency_id", rate.CurrencyID,
		"value_date", rate.ValueDate.Format("2006-01-02"),
		"rate_avg", rate.RateAverage.String(),
	)

	return nil
}

func (r *exchangeRateRepository) GetCurrencyIDs(ctx context.Context) (map[string]int64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	currencies, err := r.client.Currency.Query().
		Where(currency.ShowEQ(true)).
		All(dbCtx)
	if err != nil {
		slog.Error("Fallo al consultar catálogo de monedas en Ent", "error", err)
		return nil, fmt.Errorf("error al consultar catálogo de monedas: %w", err)
	}

	currencyMap := make(map[string]int64)
	for _, c := range currencies {
		currencyMap[c.Code] = int64(c.ID)
	}

	return currencyMap, nil
}

func (r *exchangeRateRepository) GetLatestRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	currencies, err := r.client.Currency.Query().
		Where(currency.ShowEQ(true)).
		All(dbCtx)
	if err != nil {
		return nil, fmt.Errorf("error al consultar monedas: %w", err)
	}

	var rates []domain.ExchangeRate
	for _, c := range currencies {
		latest, err := r.client.ExchangeRate.Query().
			Where(exchangerate.CurrencyID(int64(c.ID))).
			Order(ent.Desc(exchangerate.FieldValueDate)).
			First(dbCtx)

		if err == nil {
			rates = append(rates, toDomainExchangeRate(latest, c.Code))
		}
	}

	return rates, nil
}

func (r *exchangeRateRepository) GetRatesHistoryPaginated(
	ctx context.Context,
	page, limit int,
	currencyCode string,
	startDate, endDate *time.Time,
) ([]domain.ExchangeRate, int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := r.client.ExchangeRate.Query()

	if currencyCode != "" {
		c, err := r.client.Currency.Query().
			Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
			Only(dbCtx)
		if err == nil {
			query = query.Where(exchangerate.CurrencyID(int64(c.ID)))
		}
	} else {
		cIDs, err := r.client.Currency.Query().Where(currency.ShowEQ(true)).IDs(dbCtx)
		if err == nil {
			var ids []int64
			for _, id := range cIDs {
				ids = append(ids, int64(id))
			}
			query = query.Where(exchangerate.CurrencyIDIn(ids...))
		}
	}

	if startDate != nil {
		query = query.Where(exchangerate.ValueDateGTE(*startDate))
	}
	if endDate != nil {
		query = query.Where(exchangerate.ValueDateLTE(*endDate))
	}

	totalItems, err := query.Count(dbCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("error al contar historial: %w", err)
	}

	if totalItems == 0 {
		return []domain.ExchangeRate{}, 0, nil
	}

	offset := (page - 1) * limit
	entRates, err := query.
		Order(ent.Desc(exchangerate.FieldValueDate)).
		Offset(offset).
		Limit(limit).
		All(dbCtx)

	if err != nil {
		return nil, 0, fmt.Errorf("error al consultar historial: %w", err)
	}

	cMap, _ := r.GetCurrencyIDs(ctx)
	invMap := make(map[int64]string)
	for k, v := range cMap {
		invMap[v] = k
	}

	var rates []domain.ExchangeRate
	for _, er := range entRates {
		code := invMap[er.CurrencyID]
		rates = append(rates, toDomainExchangeRate(er, code))
	}

	return rates, totalItems, nil
}

func (r *exchangeRateRepository) GetLatestRate(ctx context.Context, currencyCode string) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	latest, err := r.client.ExchangeRate.Query().
		Where(exchangerate.CurrencyID(int64(c.ID))).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		First(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener la última tasa de cambio: %w", err)
	}

	rate := toDomainExchangeRate(latest, currencyCode)
	return &rate, nil
}

func (r *exchangeRateRepository) GetPreviousRate(ctx context.Context, currencyCode string, beforeDate time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	prev, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(int64(c.ID)),
			exchangerate.ValueDateLT(beforeDate),
		).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		First(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener la tasa de cambio previa: %w", err)
	}

	rate := toDomainExchangeRate(prev, currencyCode)
	return &rate, nil
}

func (r *exchangeRateRepository) GetRateByDate(ctx context.Context, currencyCode string, date time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	target, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(int64(c.ID)),
			exchangerate.ValueDateEQ(date),
		).
		First(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener la tasa de cambio por fecha: %w", err)
	}

	rate := toDomainExchangeRate(target, currencyCode)
	return &rate, nil
}

func (r *exchangeRateRepository) GetRatesHistory(ctx context.Context, currencyCode string, limit int) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, nil
	}

	entRates, err := r.client.ExchangeRate.Query().
		Where(exchangerate.CurrencyID(int64(c.ID))).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		Limit(limit).
		All(dbCtx)

	if err != nil {
		return nil, fmt.Errorf("error al consultar el historial simple de tasas de cambio: %w", err)
	}

	var rates []domain.ExchangeRate
	for _, er := range entRates {
		rates = append(rates, toDomainExchangeRate(er, currencyCode))
	}

	return rates, nil
}

func (r *exchangeRateRepository) GetLatestRateBeforeOrAt(ctx context.Context, currencyCode string, date time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	latest, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(int64(c.ID)),
			exchangerate.ValueDateLTE(date),
		).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		First(dbCtx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("error al obtener la tasa de cambio en o antes de fecha: %w", err)
	}

	rate := toDomainExchangeRate(latest, currencyCode)
	return &rate, nil
}

func (r *exchangeRateRepository) GetRatesHistoryBeforeOrAt(ctx context.Context, currencyCode string, date time.Time, limit int) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := r.client.Currency.Query().
		Where(currency.CodeEQ(currencyCode), currency.ShowEQ(true)).
		Only(dbCtx)
	if err != nil {
		return nil, nil
	}

	entRates, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyID(int64(c.ID)),
			exchangerate.ValueDateLTE(date),
		).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		Limit(limit).
		All(dbCtx)

	if err != nil {
		return nil, fmt.Errorf("error al consultar historial simple en o antes de fecha: %w", err)
	}

	var rates []domain.ExchangeRate
	for _, er := range entRates {
		rates = append(rates, toDomainExchangeRate(er, currencyCode))
	}

	return rates, nil
}

func (r *exchangeRateRepository) GetCalendarDates(ctx context.Context) ([]string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cIDs, err := r.client.Currency.Query().
		Where(currency.ShowEQ(true)).
		IDs(dbCtx)
	if err != nil {
		return nil, fmt.Errorf("error al consultar monedas visibles: %w", err)
	}

	var ids []int64
	for _, id := range cIDs {
		ids = append(ids, int64(id))
	}

	loc, _ := time.LoadLocation("America/Caracas")
	nowCaracas := time.Now().In(loc)

	entRates, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyIDIn(ids...),
			exchangerate.ValueDateLTE(nowCaracas),
		).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		All(dbCtx)

	if err != nil {
		return nil, fmt.Errorf("error al obtener fechas de calendario: %w", err)
	}

	dateSet := make(map[string]bool)
	var dates []string
	for _, er := range entRates {
		ds := er.ValueDate.Format("2006-01-02")
		if !dateSet[ds] {
			dateSet[ds] = true
			dates = append(dates, ds)
		}
	}

	if dates == nil {
		dates = []string{}
	}
	return dates, nil
}

func (r *exchangeRateRepository) UpdateRateApproval(
	ctx context.Context,
	rateID int64,
	rateFrom, rateTo, rateAverage decimal.Decimal,
	source string,
) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rateFromFloat, _ := rateFrom.Float64()
	rateToFloat, _ := rateTo.Float64()
	rateAvgFloat, _ := rateAverage.Float64()

	_, err := r.client.ExchangeRate.UpdateOneID(int(rateID)).
		SetRateFrom(rateFromFloat).
		SetRateTo(rateToFloat).
		SetRateAverage(rateAvgFloat).
		SetUpdatedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		return fmt.Errorf("tasa de cambio con ID %d no encontrada o fallo al actualizar: %w", rateID, err)
	}

	return nil
}

func (r *exchangeRateRepository) GetLast7DaysRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cIDs, err := r.client.Currency.Query().
		Where(currency.ShowEQ(true)).
		IDs(dbCtx)
	if err != nil {
		return nil, fmt.Errorf("error al consultar monedas visibles: %w", err)
	}

	var ids []int64
	for _, id := range cIDs {
		ids = append(ids, int64(id))
	}

	cutoff := time.Now().AddDate(0, 0, -7)

	entRates, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.CurrencyIDIn(ids...),
			exchangerate.ValueDateGTE(cutoff),
		).
		Order(ent.Desc(exchangerate.FieldValueDate)).
		All(dbCtx)

	if err != nil {
		return nil, fmt.Errorf("error al consultar tasas de los últimos 7 días: %w", err)
	}

	cMap, _ := r.GetCurrencyIDs(ctx)
	invMap := make(map[int64]string)
	for k, v := range cMap {
		invMap[v] = k
	}

	var rates []domain.ExchangeRate
	for _, er := range entRates {
		code := invMap[er.CurrencyID]
		rates = append(rates, toDomainExchangeRate(er, code))
	}

	return rates, nil
}

func (r *exchangeRateRepository) MarkRateNotified(ctx context.Context, rateID int64) (bool, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	er, err := r.client.ExchangeRate.Query().
		Where(
			exchangerate.IDEQ(int(rateID)),
			exchangerate.NotifiedAtIsNil(),
		).
		Only(dbCtx)

	if err != nil {
		return false, nil
	}

	_, err = r.client.ExchangeRate.UpdateOne(er).
		SetNotifiedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		return false, fmt.Errorf("error al marcar tasa como notificada: %w", err)
	}

	return true, nil
}

func (r *exchangeRateRepository) ApproveDueRates(ctx context.Context) (int64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	loc, _ := time.LoadLocation("America/Caracas")
	nowCaracas := time.Now().In(loc)

	affected, err := r.client.ExchangeRate.Update().
		Where(exchangerate.ValueDateLTE(nowCaracas)).
		SetUpdatedAt(time.Now()).
		Save(dbCtx)

	if err != nil {
		slog.Error("Fallo al auto-aprobar tasas vencidas en Ent", "error", err)
		return 0, fmt.Errorf("error al auto-aprobar tasas vencidas: %w", err)
	}

	return int64(affected), nil
}

func toDomainExchangeRate(er *ent.ExchangeRate, currencyCode string) domain.ExchangeRate {
	if er == nil {
		return domain.ExchangeRate{}
	}
	return domain.ExchangeRate{
		ID:           int64(er.ID),
		CurrencyID:   er.CurrencyID,
		CurrencyCode: currencyCode,
		RateFrom:     decimal.NewFromFloat(er.RateFrom),
		RateTo:       decimal.NewFromFloat(er.RateTo),
		RateAverage:  decimal.NewFromFloat(er.RateAverage),
		ValueDate:    er.ValueDate,
		Status:       "APPROVED",
		Source:       "SCRAPING",
		CreatedAt:    er.CreatedAt,
		UpdatedAt:    er.UpdatedAt,
	}
}
