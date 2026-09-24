package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-at-least-32-characters"

var t0 = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func newManagerAt(secret string, ttl time.Duration, now time.Time) *JWTManager {
	m := NewJWTManager(secret, ttl)
	m.now = func() time.Time { return now }
	return m
}

func TestSignAndParse(t *testing.T) {
	m := newManagerAt(testSecret, time.Hour, t0)

	token, expiresIn, err := m.Sign(7, 2)
	require.NoError(t, err)
	assert.Equal(t, time.Hour, expiresIn)

	claims, err := m.Parse(token)
	require.NoError(t, err)

	staffID, err := claims.StaffID()
	require.NoError(t, err)
	assert.Equal(t, int64(7), staffID)
	assert.Equal(t, int64(2), claims.HospitalID)
	assert.Equal(t, t0, claims.IssuedAt.Time.UTC())
	assert.Equal(t, t0.Add(time.Hour), claims.ExpiresAt.Time.UTC())
}

func TestParse_Expiry(t *testing.T) {
	token, _, err := newManagerAt(testSecret, time.Hour, t0).Sign(7, 2)
	require.NoError(t, err)

	t.Run("valid just before expiry", func(t *testing.T) {
		_, err := newManagerAt(testSecret, time.Hour, t0.Add(59*time.Minute)).Parse(token)
		assert.NoError(t, err)
	})

	t.Run("rejected after expiry", func(t *testing.T) {
		_, err := newManagerAt(testSecret, time.Hour, t0.Add(time.Hour+time.Second)).Parse(token)
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestParse_Rejects(t *testing.T) {
	m := newManagerAt(testSecret, time.Hour, t0)
	valid, _, err := m.Sign(7, 2)
	require.NoError(t, err)

	future := jwt.NewNumericDate(t0.Add(time.Hour))
	sign := func(method jwt.SigningMethod, key any, claims jwt.Claims) string {
		tok, err := jwt.NewWithClaims(method, claims).SignedString(key)
		require.NoError(t, err)
		return tok
	}
	claims := Claims{HospitalID: 2, RegisteredClaims: jwt.RegisteredClaims{Subject: "7", ExpiresAt: future}}

	parts := strings.Split(valid, ".")
	forged := sign(jwt.SigningMethodHS256, []byte("attacker-secret-that-is-32-characters"),
		Claims{HospitalID: 1, RegisteredClaims: jwt.RegisteredClaims{Subject: "7", ExpiresAt: future}})
	tampered := parts[0] + "." + strings.Split(forged, ".")[1] + "." + parts[2]

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"wrong secret", sign(jwt.SigningMethodHS256, []byte("another-secret-that-is-32-characters"), claims)},
		{"tampered payload", tampered},
		{"alg none", sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, claims)},
		{"different HMAC algorithm", sign(jwt.SigningMethodHS512, []byte(testSecret), claims)},
		{"missing exp", sign(jwt.SigningMethodHS256, []byte(testSecret),
			Claims{HospitalID: 2, RegisteredClaims: jwt.RegisteredClaims{Subject: "7"}})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := m.Parse(tt.token)
			assert.Nil(t, claims)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

func TestClaims_StaffID(t *testing.T) {
	id, err := (&Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "42"}}).StaffID()
	require.NoError(t, err)
	assert.Equal(t, int64(42), id)

	_, err = (&Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "abc"}}).StaffID()
	assert.Error(t, err)
}
