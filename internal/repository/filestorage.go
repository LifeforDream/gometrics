package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"go.uber.org/zap"

	models "github.com/LifeforDream/gometrics/internal/model"
)

type metriclist map[string]models.Metrics

// FileBackedStorage встраивает MemStorage и дополнительно сохраняет
// метрики на диск. Если interval <= 0 (см. NewFileStorage), запись
// синхронна при каждом обновлении (syncSave); иначе — периодическая,
// через SaveMetricsJob.
type FileBackedStorage struct {
	*MemStorage
	fname    string
	syncSave bool
}

type saver interface {
	save() error
}

// NewFileStorage создаёt FileBackedStorage с сохранением в файл fname.
// Если interval <= 0, каждое обновление синхронно записывается на диск;
// иначе синхронную запись должна обеспечивать периодическая SaveMetricsJob.
// Если restore == true, перед возвратом хранилище восстанавливается
// из существующего файла.
func NewFileStorage(fname string, interval int, restore bool) (*FileBackedStorage, error) {
	f := &FileBackedStorage{
		MemStorage: NewMemStorage(),
		fname:      fname,
	}
	if interval <= 0 {
		f.syncSave = true
	}
	if restore {
		if err := f.loadMetrics(); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// SetGauge обновляет метрику в памяти через MemStorage и, если задан
// синхронный режим (syncSave), сразу сохраняет весь снапшот на диск.
func (f *FileBackedStorage) SetGauge(ctx context.Context, m models.Metrics) error {
	if err := f.MemStorage.SetGauge(ctx, m); err != nil {
		return err
	}
	if f.syncSave {
		return f.save()
	}
	return nil
}

// UpdateCounter обновляет метрику в памяти через MemStorage и, если задан
// синхронный режим (syncSave), сразу сохраняет весь снапшот на диск.
func (f *FileBackedStorage) UpdateCounter(ctx context.Context, metric models.Metrics) error {
	if err := f.MemStorage.UpdateCounter(ctx, metric); err != nil {
		return err
	}
	if f.syncSave {
		return f.save()
	}
	return nil
}

// UpdateMetrics обновляет батч метрик в памяти через MemStorage и, если
// задан синхронный режим (syncSave), сразу сохраняет весь снапшот на диск.
func (f *FileBackedStorage) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	if err := f.MemStorage.UpdateMetrics(ctx, metrics); err != nil {
		return err
	}
	if f.syncSave {
		return f.save()
	}
	return nil
}

func (f *FileBackedStorage) save() error {
	return SaveMetrics(f.fname, f.MemStorage.GetAll())
}

// Close сохраняет текущий снапшот метрик на диск. Вызывается после
// завершения всех in-flight запросов при остановке сервера.
func (f *FileBackedStorage) Close() error {
	return f.save()
}

func (f *FileBackedStorage) loadMetrics() error {
	metrics := make(metriclist)
	err := metrics.load(f.fname)
	if err != nil {
		return err
	}
	f.MemStorage.SetAll(metrics)
	return nil
}

func (metrics *metriclist) save(fname string) error {
	tfile, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer tfile.Close()

	data, err := json.MarshalIndent(metrics, "", "   ")
	if err != nil {
		return err
	}

	err = os.WriteFile(tfile.Name(), data, 0666)
	if err != nil {
		return err
	}
	return os.Rename(tfile.Name(), fname)
}

func (metrics *metriclist) load(fname string) error {
	file, err := os.OpenFile(fname, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(metrics)
	if errors.Is(err, io.EOF) {
		// fresh start, file is empty
		return nil
	}
	return err
}

// SaveMetrics сериализует переданные метрики в JSON-объект и атомарно
// записывает его в файл fname (через временный файл и переименование).
func SaveMetrics(fname string, metrics map[string]models.Metrics) error {
	mtr := metriclist(metrics)
	err := mtr.save(fname)
	return err
}

// SaveMetricsJob — фоновая горутина периодической записи метрик на диск
// (вызывает s.save() каждые interval секунд); завершается по ctx.Done().
// Если interval <= 0, немедленно возвращается — синхронный режим в этом
// случае обслуживается репозиторием напрямую, а не этой джобой.
func SaveMetricsJob(ctx context.Context, interval int, s saver, logger *zap.Logger) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := s.save()
			if err != nil {
				logger.Error("Error while saving metrics", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}
