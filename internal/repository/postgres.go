package repository

import (
	"context"
	"errors"
	"time"

	models "go-metrics-and-alerts/internal/model"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) (*PostgresStorage, error) {
	return &PostgresStorage{pool: pool}, nil
}

func (p *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	return p.executeWithRetry(ctx, func(ctx context.Context) error {
		_, err := p.pool.Exec(ctx, `
			INSERT INTO gauges (id, value) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = $2
		`, name, value)
		return err
	})
}

func (p *PostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	return p.executeWithRetry(ctx, func(ctx context.Context) error {
		_, err := p.pool.Exec(ctx, `
			INSERT INTO counters (id, delta) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET delta = counters.delta + $2
		`, name, value)
		return err
	})
}

func (p *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	var value float64
	err := p.pool.QueryRow(ctx, "SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	var value int64
	err := p.pool.QueryRow(ctx, "SELECT delta FROM counters WHERE id = $1", name).Scan(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	result := make(map[string]float64)
	rows, err := p.pool.Query(ctx, "SELECT id, value FROM gauges")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return result
	}

	return result
}

func (p *PostgresStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	result := make(map[string]int64)
	rows, err := p.pool.Query(ctx, "SELECT id, delta FROM counters")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err == nil {
			result[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return result
	}

	return result
}

func (p *PostgresStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	return p.executeWithRetry(ctx, func(ctx context.Context) error {
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		for _, metric := range metrics {
			switch metric.MType {
			case models.TypeGauge:
				if metric.Value != nil {
					_, err = tx.Exec(ctx, `
						INSERT INTO gauges (id, value) VALUES ($1, $2)
						ON CONFLICT (id) DO UPDATE SET value = $2
					`, metric.ID, *metric.Value)
					if err != nil {
						return err
					}
				}
			case models.TypeCounter:
				if metric.Delta != nil {
					_, err = tx.Exec(ctx, `
						INSERT INTO counters (id, delta) VALUES ($1, $2)
						ON CONFLICT (id) DO UPDATE SET delta = counters.delta + $2
					`, metric.ID, *metric.Delta)
					if err != nil {
						return err
					}
				}
			}
		}

		return tx.Commit(ctx)
	})
}

func (p *PostgresStorage) executeWithRetry(ctx context.Context, fn func(context.Context) error) error {
	retryIntervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryIntervals[attempt-1]):
			}
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		if !p.isRetriableError(err) {
			return err
		}

		if attempt == len(retryIntervals) {
			return err
		}
	}

	return nil
}

func (p *PostgresStorage) isRetriableError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		return false
	}

	switch pgErr.Code {
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.SQLClientUnableToEstablishSQLConnection,
		pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection,
		pgerrcode.TransactionResolutionUnknown,
		pgerrcode.ProtocolViolation:
		return true
	}

	return false
}