package repository

import (
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

// An unexpected error.
var errBoom = errors.New("boom")

func uniqueViolation(constraint string) error {
	return &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: constraint}
}

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		mock.Close()
	})
	return mock
}

func sql(fragment string) string { return regexp.QuoteMeta(fragment) }

func ptr[T any](v T) *T { return &v }
