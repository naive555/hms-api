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
	GetByID(ctx context.Context, id int64) (*model.Hospital, error)
}

type hospitalRepo struct {
	db DBTX
}

func NewHospitalRepo(db DBTX) HospitalRepository {
	return &hospitalRepo{db: db}
}

const hospitalColumns = `id, code, name, his_base_url, created_at`

func (r *hospitalRepo) GetByCode(ctx context.Context, code string) (*model.Hospital, error) {
	h, err := scanHospital(r.db.QueryRow(ctx, `SELECT `+hospitalColumns+` FROM hospitals WHERE code = $1`, code))
	if err != nil {
		return nil, fmt.Errorf("get hospital by code: %w", err)
	}

	return h, nil
}

func (r *hospitalRepo) GetByID(ctx context.Context, id int64) (*model.Hospital, error) {
	h, err := scanHospital(r.db.QueryRow(ctx, `SELECT `+hospitalColumns+` FROM hospitals WHERE id = $1`, id))
	if err != nil {
		return nil, fmt.Errorf("get hospital by id: %w", err)
	}

	return h, nil
}

func scanHospital(row pgx.Row) (*model.Hospital, error) {
	var h model.Hospital

	err := row.Scan(&h.ID, &h.Code, &h.Name, &h.HISBaseURL, &h.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &h, nil
}
