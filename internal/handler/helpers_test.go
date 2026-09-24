package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/auth"
	"github.com/naive555/hms-api/internal/middleware"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/service"
	"github.com/stretchr/testify/require"
)

func init() { gin.SetMode(gin.TestMode) }

// Tokens accepted by fakeTokens, one per hospital.
const (
	tokenHospitalA = "token-hospital-a"
	tokenHospitalB = "token-hospital-b"
	hospitalAID    = int64(1)
	hospitalBID    = int64(2)
)

type fakeTokens struct{}

func (fakeTokens) Sign(int64, int64) (string, time.Duration, error) { return "", 0, nil }

func (fakeTokens) Parse(token string) (*auth.Claims, error) {
	switch token {
	case tokenHospitalA:
		return &auth.Claims{HospitalID: hospitalAID, RegisteredClaims: jwt.RegisteredClaims{Subject: "10"}}, nil
	case tokenHospitalB:
		return &auth.Claims{HospitalID: hospitalBID, RegisteredClaims: jwt.RegisteredClaims{Subject: "20"}}, nil
	}
	return nil, auth.ErrInvalidToken
}

type fakeAuthService struct {
	registerCalls int
	gotRegister   service.RegisterInput
	registerStaff *model.Staff
	registerErr   error

	loginCalls  int
	gotLogin    service.LoginInput
	loginResult *service.LoginResult
	loginErr    error
}

func (f *fakeAuthService) Register(_ context.Context, in service.RegisterInput) (*model.Staff, error) {
	f.registerCalls++
	f.gotRegister = in
	return f.registerStaff, f.registerErr
}

func (f *fakeAuthService) Login(_ context.Context, in service.LoginInput) (*service.LoginResult, error) {
	f.loginCalls++
	f.gotLogin = in
	return f.loginResult, f.loginErr
}

type fakePatientService struct {
	calls         int
	gotHospitalID int64
	gotFilter     model.PatientFilter
	result        *service.SearchResult
	err           error
}

func (f *fakePatientService) Search(_ context.Context, hospitalID int64, filter model.PatientFilter) (*service.SearchResult, error) {
	f.calls++
	f.gotHospitalID = hospitalID
	f.gotFilter = filter
	return f.result, f.err
}

// An unexpected error.
var errBoom = errors.New("boom")

func newTestRouter(authSvc AuthService, patientSvc PatientService) *gin.Engine {
	return NewRouter(NewStaffHandler(authSvc), NewPatientHandler(patientSvc), middleware.Auth(fakeTokens{}))
}

func do(t *testing.T, r http.Handler, method, target string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(b)
	default:
		raw, err := json.Marshal(b)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorBody(t *testing.T, w *httptest.ResponseRecorder) apperr.BodyError {
	t.Helper()
	var b apperr.Body
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &b), "body: %s", w.Body.String())

	return b.Error
}

func ptr[T any](v T) *T { return &v }
