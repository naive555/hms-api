package handler

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validCredentials = map[string]string{"username": "nurse01", "password": "secret123", "hospital": "hospital-a"}

func TestHealthz(t *testing.T) {
	w := do(t, newTestRouter(&fakeAuthService{}, &fakePatientService{}), http.MethodGet, "/healthz", nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestCreateStaff_Success(t *testing.T) {
	svc := &fakeAuthService{registerStaff: &model.Staff{ID: 7, HospitalID: 1, Username: "nurse01"}}
	w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create", validCredentials, "")

	require.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"id":7,"username":"nurse01","hospital":"hospital-a"}`, w.Body.String())
	assert.Equal(t, service.RegisterInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"}, svc.gotRegister)
	assert.NotContains(t, w.Body.String(), "secret123", "password must never be echoed")
}

func TestCreateStaff_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    any
		wantMsg string
	}{
		{"malformed JSON", `{"username":`, "malformed request"},
		{"empty body", `{}`, "username is required"},
		{"missing password", map[string]string{"username": "nurse01", "hospital": "hospital-a"}, "password is required"},
		{"missing hospital", map[string]string{"username": "nurse01", "password": "secret123"}, "hospital is required"},
		{"short username", map[string]string{"username": "ab", "password": "secret123", "hospital": "hospital-a"}, "username must be at least 3 characters"},
		{"long username", map[string]string{"username": strings.Repeat("a", 51), "password": "secret123", "hospital": "hospital-a"}, "username must be at most 50 characters"},
		{"short password", map[string]string{"username": "nurse01", "password": "1234567", "hospital": "hospital-a"}, "password must be at least 8 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{}
			w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create", tt.body, "")

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, apperr.BodyError{Code: "VALIDATION_ERROR", Message: tt.wantMsg}, errorBody(t, w))
			assert.Zero(t, svc.registerCalls, "service must not be called on invalid input")
		})
	}
}

func TestCreateStaff_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"unknown hospital", apperr.ErrHospitalNotFound, http.StatusNotFound, "HOSPITAL_NOT_FOUND"},
		{"username taken", apperr.ErrUsernameTaken, http.StatusConflict, "USERNAME_TAKEN"},
		{"password too long in bytes", apperr.Validation("password must be at most 72 bytes"), http.StatusBadRequest, "VALIDATION_ERROR"},
		{"unexpected error", errBoom, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{registerErr: tt.err}
			w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create", validCredentials, "")

			require.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCode, errorBody(t, w).Code)
		})
	}
}

func TestCreateStaff_InternalErrorHidesDetails(t *testing.T) {
	svc := &fakeAuthService{registerErr: fmt.Errorf("insert staff: %w", errBoom)}
	w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/create", validCredentials, "")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "internal server error", errorBody(t, w).Message)
	assert.NotContains(t, w.Body.String(), "boom")
}

func TestLogin_Success(t *testing.T) {
	svc := &fakeAuthService{loginResult: &service.LoginResult{Token: "signed.jwt.token", ExpiresIn: time.Hour}}
	w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/login", validCredentials, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"access_token":"signed.jwt.token","token_type":"Bearer","expires_in":3600}`, w.Body.String())
	assert.Equal(t, service.LoginInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"}, svc.gotLogin)
}

func TestLogin_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    any
		wantMsg string
	}{
		{"malformed JSON", `not json`, "malformed request"},
		{"missing username", map[string]string{"password": "secret123", "hospital": "hospital-a"}, "username is required"},
		{"missing password", map[string]string{"username": "nurse01", "hospital": "hospital-a"}, "password is required"},
		{"missing hospital", map[string]string{"username": "nurse01", "password": "secret123"}, "hospital is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{}
			w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/login", tt.body, "")

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, apperr.BodyError{Code: "VALIDATION_ERROR", Message: tt.wantMsg}, errorBody(t, w))
			assert.Zero(t, svc.loginCalls)
		})
	}
}

func TestLogin_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"wrong password or unknown user", apperr.ErrInvalidCredentials, http.StatusUnauthorized, "INVALID_CREDENTIALS"},
		{"unknown hospital", apperr.ErrHospitalNotFound, http.StatusNotFound, "HOSPITAL_NOT_FOUND"},
		{"unexpected error", errBoom, http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAuthService{loginErr: tt.err}
			w := do(t, newTestRouter(svc, &fakePatientService{}), http.MethodPost, "/staff/login", validCredentials, "")

			require.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantCode, errorBody(t, w).Code)
		})
	}
}
