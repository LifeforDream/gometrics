package repository

import (
	"context"
	"database/sql"
	"errors"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/utils"
)

type DbStorage struct {
	db *sql.DB
}

func NewDbStorage(db *sql.DB) *DbStorage {
	return &DbStorage{db: db}
}

func (ds *DbStorage) Ping(ctx context.Context) error {
	return ds.db.PingContext(ctx)
}

func (ds *DbStorage) GetAllSlice(ctx context.Context) ([]models.Metrics, error) {
	metrics := make([]models.Metrics, 0)
	err := utils.WithRetryPG(func() error {
		metrics = metrics[:0]
		rows, err := ds.db.QueryContext(ctx, "SELECT id, mtype, delta, value, hash FROM metrics")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			metric := models.Metrics{}
			err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value, &metric.Hash)
			if err != nil {
				return err
			}
			metrics = append(metrics, metric)
		}

		err = rows.Err()
		if err != nil {
			return err
		}
		return nil
	})
	return metrics, err
}

func (ds *DbStorage) GetMetric(ctx context.Context, name string) (models.Metrics, error) {
	var metric models.Metrics
	err := utils.WithRetryPG(func() error {
		row := ds.db.QueryRowContext(ctx, "SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1", name)
		err := row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value, &metric.Hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return myErrors.MetricNotFound
			}
			return err
		}
		return nil
	})
	return metric, err
}

func (ds *DbStorage) SetGauge(ctx context.Context, metric models.Metrics) error {
	return utils.WithRetryPG(func() error {
		tx, err := ds.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		err = ds.setGauge(ctx, tx, metric)
		if err != nil {
			return err
		}
		return tx.Commit()
	})
}

func (ds *DbStorage) setGauge(ctx context.Context, tx *sql.Tx, metric models.Metrics) error {
	query := "INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype"
	result, err := tx.ExecContext(ctx, query, metric.ID, metric.MType, nil, *metric.Value, metric.Hash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// there was ID conflict, but mtypes were different
		var existingType string
		row := tx.QueryRowContext(ctx, "SELECT mtype FROM metrics WHERE id = $1", metric.ID)
		if scanErr := row.Scan(&existingType); scanErr != nil {
			existingType = "unknown"
		}
		return myErrors.InvalidMetricType{
			ExistingType: existingType,
			NewType:      models.Gauge,
			MetricName:   metric.ID,
		}
	}
	return nil
}

func (ds *DbStorage) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	return utils.WithRetryPG(func() error {
		tx, err := ds.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		err = ds.updateCounter(ctx, tx, metric)
		if err != nil {
			return err
		}
		return tx.Commit()
	})
}

func (ds *DbStorage) updateCounter(ctx context.Context, tx *sql.Tx, metric models.Metrics) error {
	query := "INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype"
	result, err := tx.ExecContext(ctx, query, metric.ID, metric.MType, *metric.Delta, nil, metric.Hash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// there was ID conflict, but mtypes were different
		var existingType string
		row := tx.QueryRowContext(ctx, "SELECT mtype FROM metrics WHERE id = $1", metric.ID)
		if scanErr := row.Scan(&existingType); scanErr != nil {
			existingType = "unknown"
		}
		return myErrors.InvalidMetricType{
			ExistingType: existingType,
			NewType:      models.Counter,
			MetricName:   metric.ID,
		}
	}
	return nil
}

func (ds *DbStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	return utils.WithRetryPG(func() error {
		tx, err := ds.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, metric := range metrics {
			switch metric.MType {
			case models.Counter:
				err = ds.updateCounter(ctx, tx, metric)
			case models.Gauge:
				err = ds.setGauge(ctx, tx, metric)
			default:
				err = myErrors.NonexistentMetricType
			}
			if err != nil {
				return err
			}
		}
		return tx.Commit()
	})
}

func (ds *DbStorage) Close() error {
	return ds.db.Close()
}
