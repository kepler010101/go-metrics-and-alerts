package repository

import (
	"database/sql"
	"errors"
	"time"

	models "go-metrics-and-alerts/internal/model"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) (*PostgresStorage, error) {
	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) UpdateGauge(name string, value float64) error {
	return p.executeWithRetry(func() error {
		_, err := p.db.Exec(`
			INSERT INTO gauges (id, value) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = $2
		`, name, value)
		return err
	})
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) error {
	return p.executeWithRetry(func() error {
		_, err := p.db.Exec(`
			INSERT INTO counters (id, delta) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET delta = counters.delta + $2
		`, name, value)
		return err
	})
}

func (p *PostgresStorage) GetGauge(name string) (float64, bool) {
	var value float64
	err := p.executeWithRetry(func() error {
		return p.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
	})
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetCounter(name string) (int64, bool) {
	var value int64
	err := p.executeWithRetry(func() error {
		return p.db.QueryRow("SELECT delta FROM counters WHERE id = $1", name).Scan(&value)
	})
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)

	var rows *sql.Rows
	err := p.executeWithRetry(func() error {
		var err error
		rows, err = p.db.Query("SELECT id, value FROM gauges")
		return err
	})

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

	if rowsErr := rows.Err(); rowsErr != nil {
		return make(map[string]float64)
	}

	return result
}

func (p *PostgresStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)

	var rows *sql.Rows
	err := p.executeWithRetry(func() error {
		var err error
		rows, err = p.db.Query("SELECT id, delta FROM counters")
		return err
	})

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

	if rowsErr := rows.Err(); rowsErr != nil {
		return make(map[string]int64)
	}

	return result
}

func (p *PostgresStorage) UpdateBatch(metrics []models.Metrics) error {
	return p.executeWithRetry(func() error {
		tx, err := p.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		gaugeStmt, err := tx.Prepare(`
			INSERT INTO gauges (id, value) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET value = $2
		`)
		if err != nil {
			return err
		}
		defer gaugeStmt.Close()

		counterStmt, err := tx.Prepare(`
			INSERT INTO counters (id, delta) VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE SET delta = counters.delta + $2
		`)
		if err != nil {
			return err
		}
		defer counterStmt.Close()

		for _, metric := range metrics {
			switch metric.MType {
			case "gauge":
				if metric.Value != nil {
					_, err = gaugeStmt.Exec(metric.ID, *metric.Value)
					if err != nil {
						return err
					}
				}
			case "counter":
				if metric.Delta != nil {
					_, err = counterStmt.Exec(metric.ID, *metric.Delta)
					if err != nil {
						return err
					}
				}
			}
		}

		return tx.Commit()
	})
}

func (p *PostgresStorage) executeWithRetry(fn func() error) error {
	retryIntervals := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	for attempt := 0; attempt <= len(retryIntervals); attempt++ {
		if attempt > 0 {
			time.Sleep(retryIntervals[attempt-1])
		}

		err := fn()
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
