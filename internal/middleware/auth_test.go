package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-that-is-at-least-32-characters"

func init() { gin.SetMode(gin.TestMode) }

// newTestRouter mounts Auth on a route that echoes the IDs it received.
func newTestRouter(tokens auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.GET("/protected", Auth(tokens), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"staff_id": StaffID(c), "hospital_id": HospitalID(c)})
	})
	return r
}

func signToken(t *testing.T, m *auth.JWTManager, staffID, hospitalID int64) string {
	t.Helper()
	tok, _, err := m.Sign(staffID, hospitalID)
	require.NoError(t, err)
	return tok
}

// signRaw signs arbitrary claims, for cases JWTManager.Sign can't produce.
func signRaw(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(method, claims).SignedString(key)
	require.NoError(t, err)
	return tok
}

func TestAuth_ValidToken(t *testing.T) {
	m := auth.NewJWTManager(testSecret, time.Hour)
	r := newTestRouter(m)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, m, 7, 2))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"staff_id":7,"hospital_id":2}`, w.Body.String())
}

func TestAuth_SchemeIsCaseInsensitive(t *testing.T) {
	m := auth.NewJWTManager(testSecret, time.Hour)
	r := newTestRouter(m)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "bearer "+signToken(t, m, 7, 2))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_Rejects(t *testing.T) {
	m := auth.NewJWTManager(testSecret, time.Hour)
	future := jwt.NewNumericDate(time.Now().Add(time.Hour))

	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"wrong scheme", "Basic " + signToken(t, m, 7, 2)},
		{"bearer without token", "Bearer "},
		{"token without scheme", signToken(t, m, 7, 2)},
		{"garbage token", "Bearer not-a-jwt"},
		{"wrong secret", "Bearer " + signToken(t, auth.NewJWTManager("another-secret-that-is-32-characters-long", time.Hour), 7, 2)},
		{"expired", "Bearer " + signToken(t, auth.NewJWTManager(testSecret, -time.Minute), 7, 2)},
		{"alg none", "Bearer " + signRaw(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType,
			auth.Claims{HospitalID: 2, RegisteredClaims: jwt.RegisteredClaims{Subject: "7", ExpiresAt: future}})},
		{"no exp", "Bearer " + signRaw(t, jwt.SigningMethodHS256, []byte(testSecret),
			auth.Claims{HospitalID: 2, RegisteredClaims: jwt.RegisteredClaims{Subject: "7"}})},
		{"non-numeric subject", "Bearer " + signRaw(t, jwt.SigningMethodHS256, []byte(testSecret),
			auth.Claims{HospitalID: 2, RegisteredClaims: jwt.RegisteredClaims{Subject: "abc", ExpiresAt: future}})},
		{"missing hospital_id", "Bearer " + signRaw(t, jwt.SigningMethodHS256, []byte(testSecret),
			auth.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "7", ExpiresAt: future}})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			r := gin.New()
			r.GET("/protected", Auth(m), func(c *gin.Context) { handlerCalled = true })

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusUnauthorized, w.Code)
			assert.False(t, handlerCalled, "handler must not run after auth fails")

			var body apperr.Body
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "UNAUTHORIZED", body.Error.Code)
		})
	}
}
