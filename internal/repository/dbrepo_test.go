package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	models "github.com/LifeforDream/gometrics/internal/model"
	myErrors "github.com/LifeforDream/gometrics/internal/model/errors"
	"github.com/LifeforDream/gometrics/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbGetAllSlice(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(sqlmock.Sqlmock)
		want    []models.Metrics
		wantErr bool
	}{
		{
			name: "empty table",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, mtype, delta, value, hash FROM metrics").
					WillReturnRows(sqlmock.NewRows([]string{"ID", "MType", "Delta", "Value", "Hash"}))
			},
			want: []models.Metrics{},
		},
		{
			name: "gauge and counter",
			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"ID", "MType", "Delta", "Value", "Hash"}).
					AddRow("alloc", "gauge", nil, 1.25, "").
					AddRow("pollcount", "counter", 5, nil, "")
				mock.ExpectQuery("SELECT id, mtype, delta, value, hash FROM metrics").
					WillReturnRows(rows)
			},
			want: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
			},
		},
		{
			name: "query error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, mtype, delta, value, hash FROM metrics").
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "streaming error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mtype, delta, value, hash FROM metrics")).
					WillReturnRows(sqlmock.NewRows([]string{"id", "mtype", "delta", "value", "hash"}).
						AddRow("alloc", "gauge", nil, 1.25, "").
						RowError(0, errors.New("connection error")))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.setup(mock)

			got, err := NewDbStorage(db).GetAllSlice(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDbGetMetric(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		setup      func(sqlmock.Sqlmock)
		want       models.Metrics
		wantErr    bool
	}{
		{
			name:       "found gauge",
			metricName: "alloc",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1")).
					WithArgs("alloc").
					WillReturnRows(sqlmock.NewRows([]string{"ID", "MType", "Delta", "Value", "Hash"}).
						AddRow("alloc", "gauge", nil, 1.25, ""))
			},
			want: models.Metrics{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
		},
		{
			name:       "found counter",
			metricName: "pollcount",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1")).
					WithArgs("pollcount").
					WillReturnRows(sqlmock.NewRows([]string{"ID", "MType", "Delta", "Value", "Hash"}).
						AddRow("pollcount", "counter", int64(5), nil, ""))
			},
			want: models.Metrics{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
		},
		{
			name:       "not found",
			metricName: "unknown",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, mtype, delta, value, hash FROM metrics WHERE id = $1")).
					WithArgs("unknown").
					WillReturnRows(sqlmock.NewRows([]string{"ID", "MType", "Delta", "Value", "Hash"}))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.setup(mock)

			got, err := NewDbStorage(db).GetMetric(context.Background(), tt.metricName)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, myErrors.MetricNotFound))
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDbSetGauge(t *testing.T) {
	tests := []struct {
		name             string
		metric           models.Metrics
		setup            func(sqlmock.Sqlmock)
		wantErr          bool
		wantTypeConflict bool
	}{
		{
			name:   "insert new gauge",
			metric: models.Metrics{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("alloc", "gauge", nil, 1.25, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:   "update existing gauge replaces value",
			metric: models.Metrics{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 2.5)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("alloc", "gauge", nil, 2.5, "").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:   "type conflict with existing counter",
			metric: models.Metrics{ID: "pollcount", MType: models.Gauge, Value: utils.FloatPtr(t, 1.0)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("pollcount", "gauge", nil, 1.0, "").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta("SELECT mtype FROM metrics WHERE id = $1")).
					WithArgs("pollcount").
					WillReturnRows(sqlmock.NewRows([]string{"MType"}).AddRow("counter"))
				mock.ExpectRollback()
			},
			wantErr:          true,
			wantTypeConflict: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.setup(mock)

			err = NewDbStorage(db).SetGauge(context.Background(), tt.metric)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantTypeConflict {
					var typeErr myErrors.InvalidMetricType
					assert.True(t, errors.As(err, &typeErr))
				}
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDbUpdateCounter(t *testing.T) {
	tests := []struct {
		name             string
		metric           models.Metrics
		setup            func(sqlmock.Sqlmock)
		wantErr          bool
		wantTypeConflict bool
	}{
		{
			name:   "insert new counter",
			metric: models.Metrics{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 5)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("pollcount", "counter", 5, nil, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:   "accumulate existing counter",
			metric: models.Metrics{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 3)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("pollcount", "counter", 3, nil, "").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:   "type conflict with existing gauge",
			metric: models.Metrics{ID: "alloc", MType: models.Counter, Delta: utils.IntPtr(t, 1)},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype")).
					WithArgs("alloc", "counter", 1, nil, "").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta("SELECT mtype FROM metrics WHERE id = $1")).
					WithArgs("alloc").
					WillReturnRows(sqlmock.NewRows([]string{"MType"}).AddRow("gauge"))
				mock.ExpectRollback()
			},
			wantErr:          true,
			wantTypeConflict: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.setup(mock)

			err = NewDbStorage(db).UpdateCounter(context.Background(), tt.metric)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantTypeConflict {
					var typeErr myErrors.InvalidMetricType
					assert.True(t, errors.As(err, &typeErr))
				}
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDbUpdateMetrics(t *testing.T) {
	gaugeSQL := regexp.QuoteMeta(`INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value WHERE metrics.mtype = EXCLUDED.mtype`)
	counterSQL := regexp.QuoteMeta(`INSERT INTO metrics(id, mtype, delta, value, hash) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta WHERE metrics.mtype = EXCLUDED.mtype`)

	tests := []struct {
		name    string
		input   []models.Metrics
		setup   func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "success with gauge and counter",
			input: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.25)},
				{ID: "pollcount", MType: models.Counter, Delta: utils.IntPtr(t, 3)},
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(gaugeSQL).
					WithArgs("alloc", "gauge", nil, 1.25, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(counterSQL).
					WithArgs("pollcount", "counter", int64(3), nil, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "type conflict on first metric rolls back",
			input: []models.Metrics{
				{ID: "alloc", MType: models.Gauge, Value: utils.FloatPtr(t, 1.0)},
				{ID: "malloc", MType: models.Gauge, Value: utils.FloatPtr(t, 5.0)},
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(gaugeSQL).
					WithArgs("alloc", "gauge", nil, 1.0, "").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery(regexp.QuoteMeta("SELECT mtype FROM metrics WHERE id = $1")).
					WithArgs("alloc").
					WillReturnRows(sqlmock.NewRows([]string{"MType"}).AddRow("counter"))
				mock.ExpectRollback()
			},
			wantErr: true,
		},
		{
			name:  "unknown metric type rolls back",
			input: []models.Metrics{{ID: "bad", MType: "invalid"}},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			tt.setup(mock)

			err = NewDbStorage(db).UpdateMetrics(context.Background(), tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
