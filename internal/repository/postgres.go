package repository

import (
	"database/sql"

	models "go-metrics-and-alerts/internal/model"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) (*PostgresStorage, error) {
	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) UpdateGauge(name string, value float64) error {
	_, err := p.db.Exec(`
		INSERT INTO gauges (id, value) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = $2
	`, name, value)
	return err
}

func (p *PostgresStorage) UpdateCounter(name string, value int64) error {
	_, err := p.db.Exec(`
		INSERT INTO counters (id, delta) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET delta = counters.delta + $2
	`, name, value)
	return err
}

func (p *PostgresStorage) GetGauge(name string) (float64, bool) {
	var value float64
	err := p.db.QueryRow("SELECT value FROM gauges WHERE id = $1", name).Scan(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetCounter(name string) (int64, bool) {
	var value int64
	err := p.db.QueryRow("SELECT delta FROM counters WHERE id = $1", name).Scan(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *PostgresStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)
	rows, err := p.db.Query("SELECT id, value FROM gauges")
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

	err = rows.Err()
	if err != nil {
		return result
	}

	return result
}

func (p *PostgresStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)
	rows, err := p.db.Query("SELECT id, delta FROM counters")
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

	err = rows.Err()
	if err != nil {
		return result
	}

	return result
}

func (p *PostgresStorage) UpdateBatch(metrics []models.Metrics) error {
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
}
