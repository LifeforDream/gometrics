// Package logging настраивает production-логгер сервера.
package logging

import (
	"go.uber.org/zap"
)

// Initialize вызывается один раз при старте сервера: строит
// production-конфиг zap с заданным уровнем level ("debug", "info" и т.д.).
func Initialize(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return zl, nil
}
