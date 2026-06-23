package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type PgxIface interface {
	Ping(ctx context.Context) error
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Close()
}

//go:embed migrations
var migrationsFS embed.FS

type DbStorage struct {
	pool   PgxIface
	logger *zap.Logger
}

func NewDbStorage(ctx context.Context, pool PgxIface, logger *zap.Logger) (*DbStorage, error) {
	store := &DbStorage{pool: pool, logger: logger}
	return store, store.performMigrate(ctx)
}

func newDbStorageForTest(pool PgxIface) *DbStorage {
	return &DbStorage{pool: pool}
}

func (ds *DbStorage) performMigrate(ctx context.Context) error {
	return utils.WithRetryPG(ctx, func() error {
		pool, ok := ds.pool.(*pgxpool.Pool)
		if !ok {
			return fmt.Errorf("incorrect pool for migrations")
		}
		db := sql.OpenDB(stdlib.GetPoolConnector(pool))

		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			return fmt.Errorf("error creating driver: %w", err)
		}

		d, err := iofs.New(migrationsFS, "migrations")
		if err != nil {
			return fmt.Errorf("error reading FS: %w", err)
		}
		m, err := migrate.NewWithInstance("iofs", d, "pgx", driver)
		if err != nil {
			return fmt.Errorf("error creating migrator: %w", err)
		}
		defer func() {
			srcErr, dbErr := m.Close()
			if srcErr != nil {
				ds.logger.Warn("failed to close migration source", zap.Error(srcErr))
			}
			if dbErr != nil {
				ds.logger.Warn("failed to close migration db", zap.Error(dbErr))
			}
		}()
		err = m.Up()
		if err == nil || errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		if !utils.IsRetriablePgError(err) {
			ds.logger.Error("error performing migration", zap.Error(err))
			return err
		}
		return err
	})
}

func (ds *DbStorage) Ping(ctx context.Context) error {
	return ds.pool.Ping(ctx)
}

func (ds *DbStorage) GetAllSlice(ctx context.Context) ([]models.Metrics, error) {
	metrics := make([]models.Metrics, 0)
	err := utils.WithRetryPG(ctx, func() error {
		metrics = metrics[:0]
		rows, err := ds.pool.Query(ctx, "SELECT id, mtype, delta, value, hash FROM metrics")
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
	err := utils.WithRetryPG(ctx, func() error {
		row := ds.pool.QueryRow(ctx, "SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1", name)
		err := row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value, &metric.Hash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: %q", myErrors.MetricNotFound, name)
			}
			return err
		}
		return nil
	})
	return metric, err
}

func (ds *DbStorage) SetGauge(ctx context.Context, metric models.Metrics) error {
	return utils.WithRetryPG(ctx, func() error {
		tx, err := ds.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		err = ds.setGauge(ctx, tx, metric)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	})
}

func (ds *DbStorage) setGauge(ctx context.Context, tx pgx.Tx, metric models.Metrics) error {
	query := "INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype"
	result, err := tx.Exec(ctx, query, metric.ID, metric.MType, nil, *metric.Value, metric.Hash)
	if err != nil {
		return err
	}
	n := result.RowsAffected()
	if n == 0 {
		// there was ID conflict, but mtypes were different
		var existingType string
		row := tx.QueryRow(ctx, "SELECT mtype FROM metrics WHERE id = $1", metric.ID)
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
	return utils.WithRetryPG(ctx, func() error {
		tx, err := ds.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		err = ds.updateCounter(ctx, tx, metric)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	})
}

func (ds *DbStorage) updateCounter(ctx context.Context, tx pgx.Tx, metric models.Metrics) error {
	query := "INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype"
	result, err := tx.Exec(ctx, query, metric.ID, metric.MType, *metric.Delta, nil, metric.Hash)
	if err != nil {
		return err
	}
	n := result.RowsAffected()
	if n == 0 {
		// there was ID conflict, but mtypes were different
		var existingType string
		row := tx.QueryRow(ctx, "SELECT mtype FROM metrics WHERE id = $1", metric.ID)
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
	return utils.WithRetryPG(ctx, func() error {
		tx, err := ds.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		batch := &pgx.Batch{}

		for _, metric := range metrics {
			switch metric.MType {
			case models.Counter:
				batch.Queue(
					"INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype",
					metric.ID, metric.MType, *metric.Delta, nil, metric.Hash,
				)
			case models.Gauge:
				batch.Queue(
					"INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype",
					metric.ID, metric.MType, nil, *metric.Value, metric.Hash,
				)
			default:
				err = fmt.Errorf("%w: %q", myErrors.NonexistentMetricType, metric.MType)
			}
			if err != nil {
				return err
			}
		}
		err = tx.SendBatch(ctx, batch).Close()

		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	})
}

func (ds *DbStorage) Close() error {
	ds.pool.Close()
	return nil
}
