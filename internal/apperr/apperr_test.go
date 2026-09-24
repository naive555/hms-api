package apperr

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError_BodyMatchesAPIEnvelope(t *testing.T) {
	out, err := json.Marshal(ErrUsernameTaken.Body())
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":{"code":"USERNAME_TAKEN","message":"username already exists in this hospital"}}`, string(out))
}

func TestValidation(t *testing.T) {
	e := Validation("email must be a valid email address")

	assert.Equal(t, http.StatusBadRequest, e.Status)
	assert.Equal(t, "VALIDATION_ERROR", e.Code)
	assert.Equal(t, "email must be a valid email address", e.Error())
}
