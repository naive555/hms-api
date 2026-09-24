package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/naive555/hms-api/internal/model"
)

type PatientRepository interface {
	Search(ctx context.Context, hospitalID int64, f model.PatientFilter) ([]model.Patient, int, error)
	Upsert(ctx context.Context, p *model.Patient) error
}

type patientRepo struct {
	db DBTX
}

func NewPatientRepo(db DBTX) PatientRepository {
	return &patientRepo{db: db}
}

const patientColumns = `id, hospital_id, patient_hn,
      first_name_th, middle_name_th, last_name_th,
      first_name_en, middle_name_en, last_name_en,
      date_of_birth, national_id, passport_id, phone_number, email, gender,
      created_at, updated_at`

func scanPatient(row pgx.Row) (model.Patient, error) {
	var p model.Patient

	err := row.Scan(&p.ID, &p.HospitalID, &p.PatientHN,
		&p.FirstNameTH, &p.MiddleNameTH, &p.LastNameTH,
		&p.FirstNameEN, &p.MiddleNameEN, &p.LastNameEN,
		&p.DateOfBirth, &p.NationalID, &p.PassportID, &p.PhoneNumber, &p.Email, &p.Gender,
		&p.CreatedAt, &p.UpdatedAt)

	return p, err
}

func (r *patientRepo) Search(ctx context.Context, hospitalID int64, f model.PatientFilter) ([]model.Patient, int, error) {
	where, args := buildPatientWhere(hospitalID, f)

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM patients WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count patients: %w", err)
	}

	patients := make([]model.Patient, 0)
	if total == 0 {
		return patients, 0, nil
	}

	query := `SELECT ` + patientColumns + ` FROM patients WHERE ` + where +
		` ORDER BY id LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	rows, err := r.db.Query(ctx, query, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		p, err := scanPatient(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan patient: %w", err)
		}
		patients = append(patients, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate patients: %w", err)
	}

	return patients, total, nil
}

type whereBuilder struct {
	conds []string
	args  []any
}

func (w *whereBuilder) add(cond string, arg any) {
	w.args = append(w.args, arg)
	w.conds = append(w.conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(w.args))))
}

func buildPatientWhere(hospitalID int64, f model.PatientFilter) (string, []any) {
	var w whereBuilder
	w.add("hospital_id = ?", hospitalID)

	// This is for indexing.
	addName := func(th, en, value string) {
		if value != "" {
			w.add("(lower("+th+") LIKE ? OR lower("+en+") LIKE ?)", prefixPattern(value))
		}
	}
	addName("first_name_th", "first_name_en", f.FirstName)
	addName("middle_name_th", "middle_name_en", f.MiddleName)
	addName("last_name_th", "last_name_en", f.LastName)

	if f.NationalID != "" {
		w.add("national_id = ?", f.NationalID)
	}
	if f.PassportID != "" {
		w.add("passport_id = ?", f.PassportID)
	}
	if f.DateOfBirth != nil {
		w.add("date_of_birth = ?", *f.DateOfBirth)
	}
	if f.PhoneNumber != "" {
		w.add("phone_number = ?", f.PhoneNumber)
	}
	if f.Email != "" {
		w.add("lower(email) = lower(?)", f.Email)
	}

	return strings.Join(w.conds, " AND "), w.args
}

func prefixPattern(s string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(s))
	return escaped + "%"
}

func (r *patientRepo) Upsert(ctx context.Context, p *model.Patient) error {
	conflict := `(hospital_id, national_id) WHERE national_id IS NOT NULL`
	if p.NationalID == nil {
		conflict = `(hospital_id, passport_id) WHERE passport_id IS NOT NULL`
	}

	err := r.db.QueryRow(ctx, `
              INSERT INTO patients (
                      hospital_id, patient_hn,
                      first_name_th, middle_name_th, last_name_th,
                      first_name_en, middle_name_en, last_name_en,
                      date_of_birth, national_id, passport_id, phone_number, email, gender)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
              ON CONFLICT `+conflict+` DO UPDATE SET
                      patient_hn     = EXCLUDED.patient_hn,
                      first_name_th  = EXCLUDED.first_name_th,
                      middle_name_th = EXCLUDED.middle_name_th,
                      last_name_th   = EXCLUDED.last_name_th,
                      first_name_en  = EXCLUDED.first_name_en,
                      middle_name_en = EXCLUDED.middle_name_en,
                      last_name_en   = EXCLUDED.last_name_en,
                      date_of_birth  = EXCLUDED.date_of_birth,
                      national_id    = EXCLUDED.national_id,
                      passport_id    = EXCLUDED.passport_id,
                      phone_number   = EXCLUDED.phone_number,
                      email          = EXCLUDED.email,
                      gender         = EXCLUDED.gender,
                      updated_at     = now()
              RETURNING id, created_at, updated_at`,
		p.HospitalID, p.PatientHN,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.NationalID, p.PassportID, p.PhoneNumber, p.Email, p.Gender).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return fmt.Errorf("%w: %s", ErrDuplicate, pgErr.ConstraintName)
	}
	if err != nil {
		return fmt.Errorf("upsert patient: %w", err)
	}

	return nil
}
