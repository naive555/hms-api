package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/naive555/hms-api/internal/model"
)

type HospitalRepository interface {
	GetByCode(ctx context.Context, code string) (*model.Hospital, error)
}

type hospitalRepo struct {
	db DBTX
}

func NewHospitalRepo(db DBTX) HospitalRepository {
	return &hospitalRepo{db: db}
}

func (r *hospitalRepo) GetByCode(ctx context.Context, code string) (*model.Hospital, error) {
	var h model.Hospital
	err := r.db.QueryRow(ctx, `
		SELECT id, code, name, his_base_url, created_at
		FROM hospitals
		WHERE code = $1`, code).
		Scan(&h.ID, &h.Code, &h.Name, &h.HISBaseURL, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get hospital by code: %w", err)
	}
	return &h, nil
}
