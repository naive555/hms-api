package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/naive555/hms-api/internal/model"
)

const pgUniqueViolation = "23505"

type StaffRepository interface {
	Create(ctx context.Context, s *model.Staff) error
	GetByUsername(ctx context.Context, hospitalID int64, username string) (*model.Staff, error)
}

type staffRepo struct {
	db DBTX
}

func NewStaffRepo(db DBTX) StaffRepository {
	return &staffRepo{db: db}
}

func (r *staffRepo) Create(ctx context.Context, s *model.Staff) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO staff (hospital_id, username, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`,
		s.HospitalID, s.Username, s.PasswordHash).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("create staff: %w", err)
	}
	return nil
}

func (r *staffRepo) GetByUsername(ctx context.Context, hospitalID int64, username string) (*model.Staff, error) {
	var s model.Staff

	err := r.db.QueryRow(ctx, `
		SELECT id, hospital_id, username, password_hash, created_at, updated_at
		FROM staff
		WHERE hospital_id = $1 AND username = $2`, hospitalID, username).
		Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get staff by username: %w", err)
	}
	return &s, nil
}
