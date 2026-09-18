// Package config содержит функции для работы с конфигурационными файлами.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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

// ResolveConfigPath определяет путь к файлу конфигурации до разбора
// остальных флагов — ADDRESS/RESTORE и т.д. ещё не объявлены на этом этапе,
// поэтому полноценный flag.FlagSet использовать нельзя (он упадёт на первом
// же неизвестном флаге). CONFIG имеет приоритет над -c/-config, как и для
// остальных опций.
func ResolveConfigPath(args []string) string {
	if v, ok := os.LookupEnv("CONFIG"); ok && v != "" {
		return v
	}
	for i, arg := range args {
		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name != "c" && name != "config" {
			continue
		}
		if hasValue {
			return value
		}
		if i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// StrOr возвращает значение строки по указателю, если он не nil, иначе возвращает значение по умолчанию.
func StrOr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

// BoolOr возвращает значение bool по указателю, если он не nil, иначе возвращает значение по умолчанию.
func BoolOr(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}

// IntOr возвращает значение int по указателю, если он не nil, иначе возвращает значение по умолчанию.
func IntOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}
