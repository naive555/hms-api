package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/naive555/hms-api/internal/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var hospitalRowColumns = []string{"id", "code", "name", "his_base_url", "created_at"}

func TestHospitalRepo_GetByCode(t *testing.T) {
	mock := newMock(t)
	created := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sql("FROM hospitals WHERE code = $1")).
		WithArgs("hospital-a").
		WillReturnRows(pgxmock.NewRows(hospitalRowColumns).
			AddRow(int64(1), "hospital-a", "Hospital A", "http://mockhis:8081", created))

	h, err := NewHospitalRepo(mock).GetByCode(context.Background(), "hospital-a")
	require.NoError(t, err)
	assert.Equal(t, &model.Hospital{ID: 1, Code: "hospital-a", Name: "Hospital A", HISBaseURL: "http://mockhis:8081", CreatedAt: created}, h)
}

func TestHospitalRepo_GetByID(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(sql("FROM hospitals WHERE id = $1")).
		WithArgs(int64(2)).
		WillReturnRows(pgxmock.NewRows(hospitalRowColumns).
			AddRow(int64(2), "hospital-b", "Hospital B", "https://hospital-b.api.co.th", time.Now()))

	h, err := NewHospitalRepo(mock).GetByID(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, "hospital-b", h.Code)
}

func TestHospitalRepo_Errors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{"no rows maps to ErrNotFound", pgx.ErrNoRows, ErrNotFound},
		{"other errors are wrapped", errBoom, errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMock(t)
			repo := NewHospitalRepo(mock)

			mock.ExpectQuery(sql("WHERE code = $1")).WithArgs("hospital-x").WillReturnError(tt.dbErr)
			_, err := repo.GetByCode(context.Background(), "hospital-x")
			assert.ErrorIs(t, err, tt.wantErr)

			mock.ExpectQuery(sql("WHERE id = $1")).WithArgs(int64(99)).WillReturnError(tt.dbErr)
			_, err = repo.GetByID(context.Background(), 99)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
