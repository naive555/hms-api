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

func TestStaffRepo_Create(t *testing.T) {
	mock := newMock(t)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sql("INSERT INTO staff (hospital_id, username, password_hash)")).
		WithArgs(int64(1), "nurse01", "$2a$hash").
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(7), now, now))

	s := &model.Staff{HospitalID: 1, Username: "nurse01", PasswordHash: "$2a$hash"}
	require.NoError(t, NewStaffRepo(mock).Create(context.Background(), s))

	assert.Equal(t, int64(7), s.ID, "generated ID is written back")
	assert.Equal(t, now, s.CreatedAt)
	assert.Equal(t, now, s.UpdatedAt)
}

func TestStaffRepo_Create_Errors(t *testing.T) {
	tests := []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{"unique violation maps to ErrDuplicate", uniqueViolation("uq_staff_hospital_username"), ErrDuplicate},
		{"other errors are wrapped", errBoom, errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMock(t)
			mock.ExpectQuery(sql("INSERT INTO staff")).
				WithArgs(int64(1), "nurse01", pgxmock.AnyArg()).
				WillReturnError(tt.dbErr)

			err := NewStaffRepo(mock).Create(context.Background(), &model.Staff{HospitalID: 1, Username: "nurse01"})
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestStaffRepo_GetByUsername(t *testing.T) {
	mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(sql("FROM staff WHERE hospital_id = $1 AND username = $2")).
		WithArgs(int64(2), "nurse01").
		WillReturnRows(pgxmock.NewRows([]string{"id", "hospital_id", "username", "password_hash", "created_at", "updated_at"}).
			AddRow(int64(6), int64(2), "nurse01", "$2a$hash", now, now))

	s, err := NewStaffRepo(mock).GetByUsername(context.Background(), 2, "nurse01")
	require.NoError(t, err)
	assert.Equal(t, &model.Staff{ID: 6, HospitalID: 2, Username: "nurse01", PasswordHash: "$2a$hash", CreatedAt: now, UpdatedAt: now}, s)
}

func TestStaffRepo_GetByUsername_Errors(t *testing.T) {
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
			mock.ExpectQuery(sql("FROM staff")).WithArgs(int64(1), "ghost").WillReturnError(tt.dbErr)

			s, err := NewStaffRepo(mock).GetByUsername(context.Background(), 1, "ghost")
			assert.Nil(t, s)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
