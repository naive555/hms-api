package service

import (
	"context"
	"errors"
	"time"

	"github.com/naive555/hms-api/internal/auth"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/repository"
)

// An unexpected error as always
var errBoom = errors.New("boom")

func ptr[T any](v T) *T { return &v }

type fakeHospitalRepo struct {
	hospitals []model.Hospital
	err       error
}

func (f *fakeHospitalRepo) GetByCode(_ context.Context, code string) (*model.Hospital, error) {
	return f.find(func(h model.Hospital) bool { return h.Code == code })
}

func (f *fakeHospitalRepo) GetByID(_ context.Context, id int64) (*model.Hospital, error) {
	return f.find(func(h model.Hospital) bool { return h.ID == id })
}

func (f *fakeHospitalRepo) find(match func(model.Hospital) bool) (*model.Hospital, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, h := range f.hospitals {
		if match(h) {
			return &h, nil
		}
	}

	return nil, repository.ErrNotFound
}

type fakeStaffRepo struct {
	staff     []model.Staff
	createErr error
	getErr    error
	created   []model.Staff
}

func (f *fakeStaffRepo) Create(_ context.Context, s *model.Staff) error {
	if f.createErr != nil {
		return f.createErr
	}
	s.ID = int64(len(f.staff) + 1)
	f.staff = append(f.staff, *s)
	f.created = append(f.created, *s)

	return nil
}

func (f *fakeStaffRepo) GetByUsername(_ context.Context, hospitalID int64, username string) (*model.Staff, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	for _, s := range f.staff {
		if s.HospitalID == hospitalID && s.Username == username {
			return &s, nil
		}
	}

	return nil, repository.ErrNotFound
}

type fakeTokens struct {
	signErr       error
	gotStaffID    int64
	gotHospitalID int64
}

func (f *fakeTokens) Sign(staffID, hospitalID int64) (string, time.Duration, error) {
	f.gotStaffID, f.gotHospitalID = staffID, hospitalID
	if f.signErr != nil {
		return "", 0, f.signErr
	}
	return "signed-token", time.Hour, nil
}

func (f *fakeTokens) Parse(string) (*auth.Claims, error) { return nil, auth.ErrInvalidToken }

type searchCall struct {
	hospitalID int64
	filter     model.PatientFilter
}

type fakePatientRepo struct {
	results   []searchResult
	calls     []searchCall
	upserted  []model.Patient
	upsertErr error
}

type searchResult struct {
	patients []model.Patient
	total    int
	err      error
}

func (f *fakePatientRepo) Search(_ context.Context, hospitalID int64, filter model.PatientFilter) ([]model.Patient, int, error) {
	f.calls = append(f.calls, searchCall{hospitalID, filter})
	if len(f.calls) > len(f.results) {
		return []model.Patient{}, 0, nil
	}
	r := f.results[len(f.calls)-1]

	return r.patients, r.total, r.err
}

func (f *fakePatientRepo) Upsert(_ context.Context, p *model.Patient) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = append(f.upserted, *p)

	return nil
}

type fakeHIS struct {
	patient    *model.Patient
	err        error
	calls      int
	gotBaseURL string
	gotID      string
}

func (f *fakeHIS) SearchByID(_ context.Context, baseURL, id string) (*model.Patient, error) {
	f.calls++
	f.gotBaseURL, f.gotID = baseURL, id
	if f.err != nil {
		return nil, f.err
	}
	p := *f.patient

	return &p, nil
}
