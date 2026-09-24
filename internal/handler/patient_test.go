package handler

import (
	"net/http"
	"testing"

	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func somchai() model.Patient {
	dob, _ := model.ParseDate("1990-05-12")
	return model.Patient{
		ID: 1, HospitalID: hospitalAID, PatientHN: "HN0001",
		FirstNameTH: ptr("สมชาย"), LastNameTH: ptr("ใจดี"),
		FirstNameEN: ptr("Somchai"), LastNameEN: ptr("Jaidee"),
		DateOfBirth: &dob, NationalID: ptr("1234567890123"),
		PhoneNumber: ptr("0812345678"), Email: ptr("somchai@example.com"), Gender: ptr("M"),
	}
}

func TestSearchPatients_Success(t *testing.T) {
	svc := &fakePatientService{result: &service.SearchResult{Patients: []model.Patient{somchai()}, Total: 1}}
	w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search?first_name=Somchai", nil, tokenHospitalA)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
              "data": [{
                      "patient_hn": "HN0001",
                      "first_name_th": "สมชาย", "middle_name_th": null, "last_name_th": "ใจดี",
                      "first_name_en": "Somchai", "middle_name_en": null, "last_name_en": "Jaidee",
                      "date_of_birth": "1990-05-12",
                      "national_id": "1234567890123", "passport_id": null,
                      "phone_number": "0812345678", "email": "somchai@example.com",
                      "gender": "M"
              }],
              "total": 1, "limit": 20, "offset": 0
      }`, w.Body.String())
}

func TestSearchPatients_EmptyResultIsEmptyArray(t *testing.T) {
	svc := &fakePatientService{result: &service.SearchResult{Patients: []model.Patient{}, Total: 0}}
	w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search?national_id=9999999999999", nil, tokenHospitalA)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":[],"total":0,"limit":20,"offset":0}`, w.Body.String())
}

func TestSearchPatients_BuildsFilterFromQuery(t *testing.T) {
	svc := &fakePatientService{result: &service.SearchResult{Patients: []model.Patient{}}}
	target := "/patient/search?national_id=1234567890123&passport_id=AB1234567" +
		"&first_name=+Som+&middle_name=M&last_name=Jai&date_of_birth=1990-05-12" +
		"&phone_number=0812345678&email=somchai@example.com&limit=5&offset=10"
	w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, target, nil, tokenHospitalA)

	require.Equal(t, http.StatusOK, w.Code)
	dob, _ := model.ParseDate("1990-05-12")
	assert.Equal(t, model.PatientFilter{
		NationalID:  "1234567890123",
		PassportID:  "AB1234567",
		FirstName:   "Som",
		MiddleName:  "M",
		LastName:    "Jai",
		DateOfBirth: &dob,
		PhoneNumber: "0812345678",
		Email:       "somchai@example.com",
		Limit:       5,
		Offset:      10,
	}, svc.gotFilter)
}

func TestSearchPatients_NoFiltersUsesDefaults(t *testing.T) {
	svc := &fakePatientService{result: &service.SearchResult{Patients: []model.Patient{}}}
	w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search", nil, tokenHospitalA)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, model.PatientFilter{Limit: 20, Offset: 0}, svc.gotFilter)
}

func TestSearchPatients_HospitalComesFromToken(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		target         string
		wantHospitalID int64
	}{
		{"hospital A token", tokenHospitalA, "/patient/search", hospitalAID},
		{"hospital B token", tokenHospitalB, "/patient/search", hospitalBID},
		{"hospital_id param is ignored", tokenHospitalA, "/patient/search?hospital_id=2", hospitalAID},
		{"hospital param is ignored", tokenHospitalA, "/patient/search?hospital=hospital-b", hospitalAID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakePatientService{result: &service.SearchResult{Patients: []model.Patient{}}}
			w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, tt.target, nil, tt.token)

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.wantHospitalID, svc.gotHospitalID)
		})
	}
}

func TestSearchPatients_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantMsg string
	}{
		{"bad date format", "date_of_birth=12/05/1990", "date_of_birth must be in YYYY-MM-DD format"},
		{"impossible date", "date_of_birth=1990-02-30", "date_of_birth must be in YYYY-MM-DD format"},
		{"bad email", "email=not-an-email", "email must be a valid email address"},
		{"national_id too long", "national_id=12345678901234", "national_id must be at most 13 characters"},
		{"limit too large", "limit=101", "limit must be at most 100"},
		{"negative limit", "limit=-1", "limit must be at least 1"},
		{"negative offset", "offset=-1", "offset must be at least 0"},
		{"non-numeric limit", "limit=abc", "malformed request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakePatientService{}
			w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search?"+tt.query, nil, tokenHospitalA)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "VALIDATION_ERROR", errorBody(t, w).Code)
			assert.Equal(t, tt.wantMsg, errorBody(t, w).Message)
			assert.Zero(t, svc.calls, "service must not be called on invalid input")
		})
	}
}

func TestSearchPatients_Unauthorized(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"no token", ""},
		{"invalid token", "forged-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakePatientService{}
			w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search?first_name=som", nil, tt.token)

			require.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Equal(t, "UNAUTHORIZED", errorBody(t, w).Code)
			assert.Zero(t, svc.calls)
		})
	}
}

func TestSearchPatients_ServiceError(t *testing.T) {
	svc := &fakePatientService{err: errBoom}
	w := do(t, newTestRouter(&fakeAuthService{}, svc), http.MethodGet, "/patient/search?first_name=som", nil, tokenHospitalA)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL_ERROR", errorBody(t, w).Code)
}
