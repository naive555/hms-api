package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var testHospitals = []model.Hospital{
	{ID: 1, Code: "hospital-a", HISBaseURL: "http://his-a"},
	{ID: 2, Code: "hospital-b", HISBaseURL: "http://his-b"},
}

func newAuthService(staff *fakeStaffRepo, hospitals *fakeHospitalRepo, tokens *fakeTokens) *AuthService {
	return NewAuthService(hospitals, staff, tokens, bcrypt.MinCost)
}

func hashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	require.NoError(t, err)

	return string(h)
}

func TestRegister_Success(t *testing.T) {
	staff := &fakeStaffRepo{}
	svc := newAuthService(staff, &fakeHospitalRepo{hospitals: testHospitals}, &fakeTokens{})

	st, err := svc.Register(context.Background(), RegisterInput{
		Username: "  nurse01 ", Password: "secret123", HospitalCode: " hospital-b ",
	})
	require.NoError(t, err)

	assert.Equal(t, int64(1), st.ID)
	assert.Equal(t, "nurse01", st.Username, "username is trimmed")
	assert.Equal(t, int64(2), st.HospitalID, "staff belongs to the hospital from the code")
	assert.NotEqual(t, "secret123", st.PasswordHash, "password is never stored in plain text")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(st.PasswordHash), []byte("secret123")))
	require.Len(t, staff.created, 1)
}

func TestRegister_Errors(t *testing.T) {
	tests := []struct {
		name      string
		input     RegisterInput
		hospitals *fakeHospitalRepo
		staff     *fakeStaffRepo
		wantErr   error
	}{
		{
			name:      "password over 72 bytes",
			input:     RegisterInput{Username: "nurse01", Password: strings.Repeat("ก", 25), HospitalCode: "hospital-a"}, // 75 bytes
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{},
		},
		{
			name:      "unknown hospital",
			input:     RegisterInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-x"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{},
			wantErr:   apperr.ErrHospitalNotFound,
		},
		{
			name:      "hospital lookup fails",
			input:     RegisterInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{err: errBoom},
			staff:     &fakeStaffRepo{},
			wantErr:   errBoom,
		},
		{
			name:      "username taken in this hospital",
			input:     RegisterInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{createErr: fmt.Errorf("%w: uq_staff_hospital_username", repository.ErrDuplicate)},
			wantErr:   apperr.ErrUsernameTaken,
		},
		{
			name:      "insert fails",
			input:     RegisterInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{createErr: errBoom},
			wantErr:   errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAuthService(tt.staff, tt.hospitals, &fakeTokens{})

			st, err := svc.Register(context.Background(), tt.input)
			assert.Nil(t, st)
			require.Error(t, err)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				var ae *apperr.Error
				require.ErrorAs(t, err, &ae)
				assert.Equal(t, "VALIDATION_ERROR", ae.Code)
			}
			assert.Empty(t, tt.staff.created)
		})
	}
}

func TestLogin_Success(t *testing.T) {
	staff := &fakeStaffRepo{staff: []model.Staff{
		{ID: 5, HospitalID: 1, Username: "nurse01", PasswordHash: hashPassword(t, "secret123")},
	}}
	tokens := &fakeTokens{}
	svc := newAuthService(staff, &fakeHospitalRepo{hospitals: testHospitals}, tokens)

	res, err := svc.Login(context.Background(), LoginInput{Username: " nurse01 ", Password: "secret123", HospitalCode: "hospital-a"})
	require.NoError(t, err)

	assert.Equal(t, &LoginResult{Token: "signed-token", ExpiresIn: time.Hour}, res)
	assert.Equal(t, int64(5), tokens.gotStaffID)
	assert.Equal(t, int64(1), tokens.gotHospitalID, "token carries the staff member's hospital")
}

func TestLogin_SameUsernameInTwoHospitals(t *testing.T) {
	staff := &fakeStaffRepo{staff: []model.Staff{
		{ID: 5, HospitalID: 1, Username: "nurse01", PasswordHash: hashPassword(t, "password-a")},
		{ID: 6, HospitalID: 2, Username: "nurse01", PasswordHash: hashPassword(t, "password-b")},
	}}
	tokens := &fakeTokens{}
	svc := newAuthService(staff, &fakeHospitalRepo{hospitals: testHospitals}, tokens)

	_, err := svc.Login(context.Background(), LoginInput{Username: "nurse01", Password: "password-b", HospitalCode: "hospital-b"})
	require.NoError(t, err)
	assert.Equal(t, int64(6), tokens.gotStaffID)
	assert.Equal(t, int64(2), tokens.gotHospitalID)

	_, err = svc.Login(context.Background(), LoginInput{Username: "nurse01", Password: "password-a", HospitalCode: "hospital-b"})
	assert.ErrorIs(t, err, apperr.ErrInvalidCredentials, "hospital A's password must not work for hospital B")
}

func TestLogin_Errors(t *testing.T) {
	existing := []model.Staff{{ID: 5, HospitalID: 1, Username: "nurse01", PasswordHash: hashPassword(t, "secret123")}}

	tests := []struct {
		name      string
		input     LoginInput
		hospitals *fakeHospitalRepo
		staff     *fakeStaffRepo
		tokens    *fakeTokens
		wantErr   error
	}{
		{
			name:      "wrong password",
			input:     LoginInput{Username: "nurse01", Password: "wrong-password", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{staff: existing},
			wantErr:   apperr.ErrInvalidCredentials,
		},
		{
			name:      "unknown username",
			input:     LoginInput{Username: "ghost", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{staff: existing},
			wantErr:   apperr.ErrInvalidCredentials,
		},
		{
			name:      "user exists only in another hospital",
			input:     LoginInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-b"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{staff: existing},
			wantErr:   apperr.ErrInvalidCredentials,
		},
		{
			name:      "unknown hospital",
			input:     LoginInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-x"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{staff: existing},
			wantErr:   apperr.ErrHospitalNotFound,
		},
		{
			name:      "staff lookup fails",
			input:     LoginInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{getErr: errBoom},
			wantErr:   errBoom,
		},
		{
			name:      "signing fails",
			input:     LoginInput{Username: "nurse01", Password: "secret123", HospitalCode: "hospital-a"},
			hospitals: &fakeHospitalRepo{hospitals: testHospitals},
			staff:     &fakeStaffRepo{staff: existing},
			tokens:    &fakeTokens{signErr: errBoom},
			wantErr:   errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tt.tokens
			if tokens == nil {
				tokens = &fakeTokens{}
			}
			svc := newAuthService(tt.staff, tt.hospitals, tokens)

			res, err := svc.Login(context.Background(), tt.input)
			assert.Nil(t, res)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewAuthService_PanicsOnInvalidCost(t *testing.T) {
	assert.Panics(t, func() {
		NewAuthService(&fakeHospitalRepo{}, &fakeStaffRepo{}, &fakeTokens{}, bcrypt.MaxCost+1)
	})
}
