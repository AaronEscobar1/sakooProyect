package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aaron/sakoo-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// exchangeRateRepository implementa la interfaz domain.ExchangeRateRepository para PostgreSQL.
type exchangeRateRepository struct {
	db *pgxpool.Pool
}

// NewExchangeRateRepository crea una nueva instancia del repositorio de tasas de cambio.
func NewExchangeRateRepository(db *pgxpool.Pool) domain.ExchangeRateRepository {
	return &exchangeRateRepository{
		db: db,
	}
}

// Upsert inserta o actualiza la tasa de cambio de forma atómica e idempotente.
func (r *exchangeRateRepository) Upsert(ctx context.Context, rate *domain.ExchangeRate) error {
	// Definición de un timeout seguro de base de datos (5 segundos) heredado del contexto padre
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Preparando Upsert de tasa de cambio", 
		"currency_id", rate.CurrencyID, 
		"value_date", rate.ValueDate,
	)

	// Consulta SQL idempotente: Si ya existe una tasa para esa moneda y fecha, se actualizan los valores
	query := `
		INSERT INTO public.exchange_rates (
			currency_id, 
			rate_from, 
			rate_to, 
			rate_average, 
			value_date, 
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (currency_id, value_date) 
		DO UPDATE SET
			rate_from = EXCLUDED.rate_from,
			rate_to = EXCLUDED.rate_to,
			rate_average = EXCLUDED.rate_average,
			updated_at = NOW()
		RETURNING id;
	`

	// Ejecución parametrizada segura y captura del ID generado o actualizado
	err := r.db.QueryRow(dbCtx, query,
		rate.CurrencyID,
		rate.RateFrom,
		rate.RateTo,
		rate.RateAverage,
		rate.ValueDate,
	).Scan(&rate.ID)

	if err != nil {
		slog.Error("Fallo al ejecutar Upsert de tasa de cambio en PostgreSQL",
			"error", err,
			"currency_id", rate.CurrencyID,
			"value_date", rate.ValueDate,
		)
		return fmt.Errorf("error al persistir tasa de cambio (upsert): %w", err)
	}

	slog.Info("Tasa de cambio persistida correctamente (Upsert exitoso)",
		"id", rate.ID,
		"currency_id", rate.CurrencyID,
		"value_date", rate.ValueDate.Format("2006-01-02"),
		"rate_avg", rate.RateAverage.String(),
	)

	return nil
}

// GetCurrencyIDs obtiene todos los códigos de moneda y sus IDs correspondientes desde catalogs.currency.
func (r *exchangeRateRepository) GetCurrencyIDs(ctx context.Context) (map[string]int64, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando catálogo de monedas en la base de datos")

	query := "SELECT id, code FROM catalogs.currency"
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al consultar catalogs.currency en PostgreSQL", "error", err)
		return nil, fmt.Errorf("error al consultar catálogo de monedas: %w", err)
	}
	defer rows.Close()

	currencyMap := make(map[string]int64)
	for rows.Next() {
		var id int64
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			slog.Error("Fallo al escanear fila de catalogs.currency", "error", err)
			return nil, fmt.Errorf("error al escanear moneda: %w", err)
		}
		currencyMap[code] = id
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de filas de catálogo de monedas: %w", err)
	}

	slog.Info("Catálogo de monedas cargado exitosamente", "count", len(currencyMap))
	return currencyMap, nil
}

// GetLatestRates obtiene la última tasa reportada para cada moneda.
func (r *exchangeRateRepository) GetLatestRates(ctx context.Context) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Obtiene solo el último registro por cada moneda basado en la fecha valor
	query := `
		SELECT DISTINCT ON (e.currency_id) 
			e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		ORDER BY e.currency_id, e.value_date DESC;
	`
	
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		slog.Error("Fallo al consultar últimas tasas de cambio en PostgreSQL", "error", err)
		return nil, fmt.Errorf("error al consultar últimas tasas: %w", err)
	}
	defer rows.Close()

	var rates []domain.ExchangeRate
	for rows.Next() {
		var rate domain.ExchangeRate
		if err := rows.Scan(
			&rate.ID,
			&rate.CurrencyID,
			&rate.CurrencyCode,
			&rate.RateFrom,
			&rate.RateTo,
			&rate.RateAverage,
			&rate.ValueDate,
			&rate.UpdatedAt,
		); err != nil {
			slog.Error("Fallo al escanear fila de exchange_rates", "error", err)
			return nil, fmt.Errorf("error al escanear tasa de cambio: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración de tasas de cambio: %w", err)
	}

	slog.Info("Últimas tasas de cambio obtenidas exitosamente", "count", len(rates))
	return rates, nil
}

// GetRatesHistoryPaginated obtiene el historial paginado y filtrado de tasas de cambio.
func (r *exchangeRateRepository) GetRatesHistoryPaginated(
	ctx context.Context, 
	page, limit int, 
	currencyCode string, 
	startDate, endDate *time.Time,
) ([]domain.ExchangeRate, int, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Iniciando consulta de historial de tasas en base de datos", 
		"page", page, 
		"limit", limit, 
		"currency_code", currencyCode,
	)

	// Construcción dinámica de condiciones WHERE y argumentos
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if currencyCode != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("c.code = $%d", argIndex))
		args = append(args, currencyCode)
		argIndex++
	}

	if startDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.value_date >= $%d", argIndex))
		args = append(args, *startDate)
		argIndex++
	}

	if endDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.value_date <= $%d", argIndex))
		args = append(args, *endDate)
		argIndex++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 1. Obtener el total de elementos
	countQuery := `
		SELECT COUNT(*)
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
	` + whereSQL

	var totalItems int
	err := r.db.QueryRow(dbCtx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		slog.Error("Fallo al obtener recuento de historial de tasas de cambio", "error", err, "query", countQuery)
		return nil, 0, fmt.Errorf("error al contar historial: %w", err)
	}

	if totalItems == 0 {
		return []domain.ExchangeRate{}, 0, nil
	}

	// 2. Obtener los elementos con paginación LIMIT / OFFSET
	offset := (page - 1) * limit

	dataArgs := append([]interface{}{}, args...)
	limitIndex := len(dataArgs) + 1
	dataArgs = append(dataArgs, limit)
	offsetIndex := len(dataArgs) + 1
	dataArgs = append(dataArgs, offset)

	dataQuery := fmt.Sprintf(`
		SELECT 
			e.id, 
			e.currency_id, 
			c.code, 
			e.rate_from, 
			e.rate_to, 
			e.rate_average, 
			e.value_date, 
			e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		%s
		ORDER BY e.value_date DESC, c.code ASC
		LIMIT $%d OFFSET $%d;
	`, whereSQL, limitIndex, offsetIndex)

	rows, err := r.db.Query(dbCtx, dataQuery, dataArgs...)
	if err != nil {
		slog.Error("Fallo al consultar historial paginado de tasas de cambio", "error", err, "query", dataQuery)
		return nil, 0, fmt.Errorf("error al consultar historial: %w", err)
	}
	defer rows.Close()

	var rates []domain.ExchangeRate
	for rows.Next() {
		var rate domain.ExchangeRate
		if err := rows.Scan(
			&rate.ID,
			&rate.CurrencyID,
			&rate.CurrencyCode,
			&rate.RateFrom,
			&rate.RateTo,
			&rate.RateAverage,
			&rate.ValueDate,
			&rate.UpdatedAt,
		); err != nil {
			slog.Error("Fallo al escanear fila de historial de exchange_rates", "error", err)
			return nil, 0, fmt.Errorf("error al escanear tasa de cambio: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error en iteración del historial de tasas: %w", err)
	}

	slog.Info("Historial de tasas de cambio obtenido exitosamente", "count", len(rates), "total", totalItems)
	return rates, totalItems, nil
}

// GetLatestRate obtiene la última tasa de cambio para una moneda específica.
func (r *exchangeRateRepository) GetLatestRate(ctx context.Context, currencyCode string) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando última tasa de cambio para moneda", "currency_code", currencyCode)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1
		ORDER BY e.value_date DESC
		LIMIT 1;
	`

	var rate domain.ExchangeRate
	err := r.db.QueryRow(dbCtx, query, currencyCode).Scan(
		&rate.ID,
		&rate.CurrencyID,
		&rate.CurrencyCode,
		&rate.RateFrom,
		&rate.RateTo,
		&rate.RateAverage,
		&rate.ValueDate,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener la última tasa de cambio: %w", err)
	}

	return &rate, nil
}

// GetPreviousRate obtiene la tasa de cambio de la fecha hábil anterior a la fecha provista.
func (r *exchangeRateRepository) GetPreviousRate(ctx context.Context, currencyCode string, beforeDate time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando tasa de cambio previa a fecha", "currency_code", currencyCode, "before_date", beforeDate)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1 AND e.value_date < $2
		ORDER BY e.value_date DESC
		LIMIT 1;
	`

	var rate domain.ExchangeRate
	err := r.db.QueryRow(dbCtx, query, currencyCode, beforeDate).Scan(
		&rate.ID,
		&rate.CurrencyID,
		&rate.CurrencyCode,
		&rate.RateFrom,
		&rate.RateTo,
		&rate.RateAverage,
		&rate.ValueDate,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener la tasa de cambio previa: %w", err)
	}

	return &rate, nil
}

// GetRateByDate obtiene la tasa de cambio para una moneda en una fecha específica.
func (r *exchangeRateRepository) GetRateByDate(ctx context.Context, currencyCode string, date time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando tasa de cambio por fecha", "currency_code", currencyCode, "date", date)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1 AND e.value_date::date = $2::date
		ORDER BY e.value_date DESC
		LIMIT 1;
	`

	var rate domain.ExchangeRate
	err := r.db.QueryRow(dbCtx, query, currencyCode, date).Scan(
		&rate.ID,
		&rate.CurrencyID,
		&rate.CurrencyCode,
		&rate.RateFrom,
		&rate.RateTo,
		&rate.RateAverage,
		&rate.ValueDate,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener la tasa de cambio por fecha: %w", err)
	}

	return &rate, nil
}

// GetRatesHistory obtiene las últimas N tasas de cambio para una moneda específica (usado en los gráficos).
func (r *exchangeRateRepository) GetRatesHistory(ctx context.Context, currencyCode string, limit int) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de tasas simple", "currency_code", currencyCode, "limit", limit)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1
		ORDER BY e.value_date DESC
		LIMIT $2;
	`

	rows, err := r.db.Query(dbCtx, query, currencyCode, limit)
	if err != nil {
		return nil, fmt.Errorf("error al consultar el historial simple de tasas de cambio: %w", err)
	}
	defer rows.Close()

	var rates []domain.ExchangeRate
	for rows.Next() {
		var rate domain.ExchangeRate
		if err := rows.Scan(
			&rate.ID,
			&rate.CurrencyID,
			&rate.CurrencyCode,
			&rate.RateFrom,
			&rate.RateTo,
			&rate.RateAverage,
			&rate.ValueDate,
			&rate.CreatedAt,
			&rate.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear tasa de cambio del historial simple: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración del historial simple de tasas: %w", err)
	}

	return rates, nil
}

// GetLatestRateBeforeOrAt obtiene la última tasa de cambio para una moneda específica en o antes de la fecha dada.
func (r *exchangeRateRepository) GetLatestRateBeforeOrAt(ctx context.Context, currencyCode string, date time.Time) (*domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando última tasa de cambio en o antes de fecha", "currency_code", currencyCode, "date", date)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1 AND e.value_date <= $2
		ORDER BY e.value_date DESC
		LIMIT 1;
	`

	var rate domain.ExchangeRate
	err := r.db.QueryRow(dbCtx, query, currencyCode, date).Scan(
		&rate.ID,
		&rate.CurrencyID,
		&rate.CurrencyCode,
		&rate.RateFrom,
		&rate.RateTo,
		&rate.RateAverage,
		&rate.ValueDate,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error al obtener la tasa de cambio en o antes de fecha: %w", err)
	}

	return &rate, nil
}

// GetRatesHistoryBeforeOrAt obtiene las últimas N tasas de cambio para una moneda en o antes de la fecha dada.
func (r *exchangeRateRepository) GetRatesHistoryBeforeOrAt(ctx context.Context, currencyCode string, date time.Time, limit int) ([]domain.ExchangeRate, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando historial de tasas simple en o antes de fecha", "currency_code", currencyCode, "date", date, "limit", limit)

	query := `
		SELECT e.id, e.currency_id, c.code, e.rate_from, e.rate_to, e.rate_average, e.value_date, e.created_at, e.updated_at
		FROM public.exchange_rates e
		JOIN catalogs.currency c ON e.currency_id = c.id
		WHERE c.code = $1 AND e.value_date <= $2
		ORDER BY e.value_date DESC
		LIMIT $3;
	`

	rows, err := r.db.Query(dbCtx, query, currencyCode, date, limit)
	if err != nil {
		return nil, fmt.Errorf("error al consultar historial simple en o antes de fecha: %w", err)
	}
	defer rows.Close()

	var rates []domain.ExchangeRate
	for rows.Next() {
		var rate domain.ExchangeRate
		if err := rows.Scan(
			&rate.ID,
			&rate.CurrencyID,
			&rate.CurrencyCode,
			&rate.RateFrom,
			&rate.RateTo,
			&rate.RateAverage,
			&rate.ValueDate,
			&rate.CreatedAt,
			&rate.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al escanear tasa de cambio del historial simple: %w", err)
		}
		rates = append(rates, rate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error en iteración del historial simple de tasas: %w", err)
	}

	return rates, nil
}

// GetCalendarDates obtiene la lista de fechas únicas con tasas de cambio registradas, ordenadas de forma descendente.
func (r *exchangeRateRepository) GetCalendarDates(ctx context.Context) ([]string, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Debug("Consultando fechas únicas de calendario de tasas")

	query := `
		SELECT DISTINCT value_date 
		FROM public.exchange_rates 
		ORDER BY value_date DESC;
	`
	rows, err := r.db.Query(dbCtx, query)
	if err != nil {
		return nil, fmt.Errorf("error al obtener fechas de calendario: %w", err)
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, fmt.Errorf("error al escanear fecha de calendario: %w", err)
		}
		dates = append(dates, date.Format("2006-01-02"))
	}

	if dates == nil {
		dates = []string{}
	}
	return dates, nil
}
