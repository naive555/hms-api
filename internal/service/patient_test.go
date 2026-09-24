package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/naive555/hms-api/internal/his"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hospitalA = int64(1)

func somchai() model.Patient {
	return model.Patient{ID: 1, HospitalID: hospitalA, PatientHN: "HN0001", FirstNameEN: ptr("Somchai"), NationalID: ptr("1234567890123")}
}

func wichaiFromHIS() *model.Patient {
	return &model.Patient{PatientHN: "HN9001", FirstNameEN: ptr("Wichai"), NationalID: ptr("1100500077777")}
}

func newPatientService(patients *fakePatientRepo, hisClient *fakeHIS) *PatientService {
	return NewPatientService(&fakeHospitalRepo{hospitals: testHospitals}, patients, hisClient)
}

func TestSearch_LocalHit(t *testing.T) {
	patients := &fakePatientRepo{results: []searchResult{{patients: []model.Patient{somchai()}, total: 1}}}
	hisClient := &fakeHIS{}
	svc := newPatientService(patients, hisClient)

	filter := model.PatientFilter{NationalID: "1234567890123", Limit: 20}
	res, err := svc.Search(context.Background(), hospitalA, filter)
	require.NoError(t, err)

	assert.Equal(t, &SearchResult{Patients: []model.Patient{somchai()}, Total: 1}, res)
	assert.Equal(t, []searchCall{{hospitalA, filter}}, patients.calls, "searched once, scoped to the staff's hospital")
	assert.Zero(t, hisClient.calls, "HIS is not called when the patient is found locally")
}

func TestSearch_NoMatchWithoutIDSkipsHIS(t *testing.T) {
	patients := &fakePatientRepo{}
	hisClient := &fakeHIS{}
	svc := newPatientService(patients, hisClient)

	res, err := svc.Search(context.Background(), hospitalA, model.PatientFilter{FirstName: "nobody", Limit: 20})
	require.NoError(t, err)

	assert.Equal(t, 0, res.Total)
	assert.Empty(t, res.Patients)
	assert.Zero(t, hisClient.calls, "the HIS can only look up by ID")
}

func TestSearch_FetchesFromHISAndSaves(t *testing.T) {
	saved := model.Patient{ID: 9, HospitalID: hospitalA, PatientHN: "HN9001", FirstNameEN: ptr("Wichai"), NationalID: ptr("1100500077777")}
	patients := &fakePatientRepo{results: []searchResult{
		{patients: []model.Patient{}, total: 0},      // first search: not in DB yet
		{patients: []model.Patient{saved}, total: 1}, // re-search after upsert
	}}
	hisClient := &fakeHIS{patient: wichaiFromHIS()}
	svc := newPatientService(patients, hisClient)

	filter := model.PatientFilter{NationalID: "1100500077777", Limit: 20}
	res, err := svc.Search(context.Background(), hospitalA, filter)
	require.NoError(t, err)

	assert.Equal(t, &SearchResult{Patients: []model.Patient{saved}, Total: 1}, res)
	assert.Equal(t, "http://his-a", hisClient.gotBaseURL, "uses the staff hospital's HIS")
	assert.Equal(t, "1100500077777", hisClient.gotID)
	require.Len(t, patients.upserted, 1)
	assert.Equal(t, hospitalA, patients.upserted[0].HospitalID, "saved under the staff's hospital")
	assert.Equal(t, []searchCall{{hospitalA, filter}, {hospitalA, filter}}, patients.calls, "re-searched with the same filter")
}

func TestSearch_HISPatientAlwaysSavedUnderStaffHospital(t *testing.T) {
	fromHIS := wichaiFromHIS()
	fromHIS.HospitalID = 2
	patients := &fakePatientRepo{}
	svc := newPatientService(patients, &fakeHIS{patient: fromHIS})

	_, err := svc.Search(context.Background(), hospitalA, model.PatientFilter{NationalID: "1100500077777", Limit: 20})
	require.NoError(t, err)

	require.Len(t, patients.upserted, 1)
	assert.Equal(t, hospitalA, patients.upserted[0].HospitalID)
}

func TestSearch_HISLookupID(t *testing.T) {
	tests := []struct {
		name   string
		filter model.PatientFilter
		wantID string
	}{
		{"national ID", model.PatientFilter{NationalID: "1100500077777"}, "1100500077777"},
		{"passport ID", model.PatientFilter{PassportID: "CD9876543"}, "CD9876543"},
		{"both prefer national ID", model.PatientFilter{NationalID: "1100500077777", PassportID: "CD9876543"}, "1100500077777"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hisClient := &fakeHIS{err: his.ErrNotFound}
			svc := newPatientService(&fakePatientRepo{}, hisClient)

			_, err := svc.Search(context.Background(), hospitalA, tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, hisClient.gotID)
		})
	}
}

func TestSearch_HISProblemsDegradeGracefully(t *testing.T) {
	tests := []struct {
		name      string
		hospitals *fakeHospitalRepo
		his       *fakeHIS
		upsertErr error
	}{
		{"HIS says not found", &fakeHospitalRepo{hospitals: testHospitals}, &fakeHIS{err: his.ErrNotFound}, nil},
		{"HIS unavailable", &fakeHospitalRepo{hospitals: testHospitals}, &fakeHIS{err: fmt.Errorf("%w: timeout", his.ErrUnavailable)}, nil},
		{"hospital lookup fails", &fakeHospitalRepo{err: errBoom}, &fakeHIS{patient: wichaiFromHIS()}, nil},
		{"HN clashes with another patient", &fakeHospitalRepo{hospitals: testHospitals}, &fakeHIS{patient: wichaiFromHIS()},
			fmt.Errorf("%w: uq_patients_hospital_hn", repository.ErrDuplicate)},
		{"upsert fails", &fakeHospitalRepo{hospitals: testHospitals}, &fakeHIS{patient: wichaiFromHIS()}, errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patients := &fakePatientRepo{upsertErr: tt.upsertErr}
			svc := NewPatientService(tt.hospitals, patients, tt.his)

			res, err := svc.Search(context.Background(), hospitalA, model.PatientFilter{NationalID: "1100500077777", Limit: 20})
			require.NoError(t, err)

			assert.Equal(t, &SearchResult{Patients: []model.Patient{}, Total: 0}, res)
			assert.Len(t, patients.calls, 1, "no re-search when nothing was saved")
		})
	}
}

func TestSearch_RepositoryErrors(t *testing.T) {
	t.Run("first search fails", func(t *testing.T) {
		patients := &fakePatientRepo{results: []searchResult{{err: errBoom}}}
		hisClient := &fakeHIS{}
		svc := newPatientService(patients, hisClient)

		res, err := svc.Search(context.Background(), hospitalA, model.PatientFilter{NationalID: "1100500077777"})
		assert.Nil(t, res)
		assert.ErrorIs(t, err, errBoom)
		assert.Zero(t, hisClient.calls)
	})

	t.Run("re-search after HIS sync fails", func(t *testing.T) {
		patients := &fakePatientRepo{results: []searchResult{{patients: []model.Patient{}}, {err: errBoom}}}
		svc := newPatientService(patients, &fakeHIS{patient: wichaiFromHIS()})

		res, err := svc.Search(context.Background(), hospitalA, model.PatientFilter{NationalID: "1100500077777"})
		assert.Nil(t, res)
		assert.ErrorIs(t, err, errBoom)
	})
}
