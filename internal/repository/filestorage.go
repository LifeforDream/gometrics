package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	models "github.com/LifeforDream/gometrics/internal/model"
	"go.uber.org/zap"
)

type metriclist map[string]models.Metrics

type FileBackedStorage struct {
	*MemStorage
	fname    string
	syncSave bool
}

type saver interface {
	Save() error
}

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

func (f *FileBackedStorage) SetGauge(m models.Metrics) error {
	if err := f.MemStorage.SetGauge(m); err != nil {
		return err
	}
	if f.syncSave {
		return f.Save()
	}
	return nil
}

func (f *FileBackedStorage) UpdateCounter(metric models.Metrics) error {
	if err := f.MemStorage.UpdateCounter(metric); err != nil {
		return err
	}
	if f.syncSave {
		return f.Save()
	}
	return nil
}

func (f *FileBackedStorage) Save() error {
	return SaveMetrics(f.fname, f.MemStorage.GetAll())
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

func SaveMetrics(fname string, metrics map[string]models.Metrics) error {
	mtr := metriclist(metrics)
	err := mtr.save(fname)
	return err
}

func SaveMetricsJob(ctx context.Context, interval int, s saver, logger *zap.Logger) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := s.Save()
			if err != nil {
				logger.Error("Error while saving metrics", zap.Error(err))
			}
		case <-ctx.Done():
			return
		}
	}
}
