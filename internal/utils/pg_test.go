package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRetry(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryDelays = orig })

	pgConnErr := &pgconn.PgError{Code: pgerrcode.ConnectionException}
	plainErr := errors.New("plain error")

	tests := []struct {
		name      string
		responses []error
		wantCalls int
		wantErr   bool
		wantErrIs error
	}{
		{
			name:      "success on first attempt",
			responses: []error{nil},
			wantCalls: 1,
		},
		{
			name:      "non-retriable error returns immediately without retrying",
			responses: []error{plainErr},
			wantCalls: 1,
			wantErr:   true,
			wantErrIs: plainErr,
		},
		{
			name:      "retriable error exhausts all three retries",
			responses: []error{pgConnErr, pgConnErr, pgConnErr, pgConnErr},
			wantCalls: 4,
			wantErr:   true,
			wantErrIs: pgConnErr,
		},
		{
			name:      "retriable error resolves on second attempt",
			responses: []error{pgConnErr, nil},
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			err := WithRetryPG(t.Context(), func() error {
				e := tt.responses[calls]
				calls++
				return e
			})

			assert.Equal(t, tt.wantCalls, calls)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsRetriablePgError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"connection exception (08000)", &pgconn.PgError{Code: pgerrcode.ConnectionException}, true},
		{"connection does not exist (08003)", &pgconn.PgError{Code: pgerrcode.ConnectionDoesNotExist}, true},
		{"connection failure (08006)", &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, true},
		{"cannot connect now (57P03)", &pgconn.PgError{Code: pgerrcode.CannotConnectNow}, true},
		{"transaction rollback (40000)", &pgconn.PgError{Code: pgerrcode.TransactionRollback}, true},
		{"serialization failure (40001)", &pgconn.PgError{Code: pgerrcode.SerializationFailure}, true},
		{"deadlock detected (40P01)", &pgconn.PgError{Code: pgerrcode.DeadlockDetected}, true},
		{"unique violation (23505)", &pgconn.PgError{Code: pgerrcode.UniqueViolation}, false},
		{"plain non-pg error", errors.New("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetriablePgError(tt.err))
		})
	}
}
