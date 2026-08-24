package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var retryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// WithRetryPG выполняет op и, если ошибка распознана IsRetriablePgError
// как транзиентная, повторяет вызов с задержками из retryDelays
// (1с, 3с, 5с). Прерывается раньше по ctx.Done(). Возвращает обёрнутую
// ошибку последней неудачной попытки.
func WithRetryPG(ctx context.Context, op func() error) error {
	err := op()
	for _, d := range retryDelays {
		if err == nil || !IsRetriablePgError(err) {
			break
		}
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
		err = op()
	}
	if err != nil {
		return fmt.Errorf("operation failed: %w", err)
	}
	return nil
}

// IsRetriablePgError определяет, стоит ли повторять операцию после err:
// true для ошибок соединения (класс 08), отката транзакции/сериализации/
// дедлока (класс 40) и "cannot connect now" (57P03); для прочих pgconn.PgError
// и любых остальных ошибок — false.
func IsRetriablePgError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		// Класс 08 - Ошибки соединения
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure:
			return true

		// Класс 40 - Откат транзакции
		case pgerrcode.TransactionRollback, // 40000
			pgerrcode.SerializationFailure, // 40001
			pgerrcode.DeadlockDetected:     // 40P01
			return true

		// Класс 57 - Ошибка оператора
		case pgerrcode.CannotConnectNow: // 57P03
			return true
		default:
			return false
		}
	}
	return false
}
