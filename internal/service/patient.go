package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/naive555/hms-api/internal/his"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/repository"
)

type PatientService struct {
	hospitals repository.HospitalRepository
	patients  repository.PatientRepository
	his       his.Client
}

func NewPatientService(
	hospitals repository.HospitalRepository,
	patients repository.PatientRepository,
	hisClient his.Client,
) *PatientService {
	return &PatientService{hospitals: hospitals, patients: patients, his: hisClient}
}

type SearchResult struct {
	Patients []model.Patient
	Total    int
}

func (s *PatientService) Search(ctx context.Context, hospitalID int64, f model.PatientFilter) (*SearchResult, error) {
	patients, total, err := s.patients.Search(ctx, hospitalID, f)
	if err != nil {
		return nil, err
	}

	if total == 0 && f.HISLookupID() != "" && s.syncFromHIS(ctx, hospitalID, f.HISLookupID()) {
		patients, total, err = s.patients.Search(ctx, hospitalID, f)
		if err != nil {
			return nil, err
		}
	}

	return &SearchResult{Patients: patients, Total: total}, nil
}

func (s *PatientService) syncFromHIS(ctx context.Context, hospitalID int64, id string) bool {
	log := slog.With("hospital_id", hospitalID)

	hospital, err := s.hospitals.GetByID(ctx, hospitalID)
	if err != nil {
		log.ErrorContext(ctx, "his sync: load hospital", "error", err)
		return false
	}

	p, err := s.his.SearchByID(ctx, hospital.HISBaseURL, id)
	switch {
	case errors.Is(err, his.ErrNotFound):
		return false
	case err != nil:
		log.WarnContext(ctx, "his sync: HIS unavailable", "error", err)
		return false
	}

	p.HospitalID = hospitalID
	if err := s.patients.Upsert(ctx, p); err != nil {
		// Duplicated, Keep local data.
		log.WarnContext(ctx, "his sync: patient not saved", "patient_hn", p.PatientHN, "error", err)
		return false
	}

	log.InfoContext(ctx, "his sync: patient saved", "patient_id", p.ID, "patient_hn", p.PatientHN)
	return true
}
