// Package config содержит функции для работы с конфигурационными файлами.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// ReadConfigFile читает JSON-конфигурацию из файла и декодирует её в структуру типа T.
func ReadConfigFile[T any](filepath string) (*T, error) {
	var jsonOpts T
	file, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file %s does not exist: %w", filepath, err)
		}
		return nil, fmt.Errorf("unexpected error opening config file %s: %w", filepath, err)
	}
	defer file.Close()
	dec := json.NewDecoder(file)

	if err := dec.Decode(&jsonOpts); err != nil {
		return nil, fmt.Errorf("error decoding json from file: %w", err)
	}
	return &jsonOpts, nil
}
