package his

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validPatientJSON = `{
      "first_name_th": "วิชัย", "middle_name_th": "", "last_name_th": "มั่นคง",
      "first_name_en": "Wichai", "middle_name_en": null, "last_name_en": "Mankong",
      "date_of_birth": "1988-03-14", "patient_hn": "HN9001",
      "national_id": "1100500077777", "passport_id": "",
      "phone_number": "0871234567", "email": "wichai@example.com", "gender": "m"
}`

// newHIS starts a fake HIS that answers every request with status and body.
func newHIS(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestSearchByID_OK(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(validPatientJSON))
	}))
	defer srv.Close()

	p, err := NewHTTPClient(time.Second).SearchByID(context.Background(), srv.URL+"/", "1100500077777")
	require.NoError(t, err)

	assert.Equal(t, "/patient/search/1100500077777", gotPath, "trailing slash in base URL is trimmed")
	assert.Equal(t, "HN9001", p.PatientHN)
	assert.Equal(t, "Wichai", *p.FirstNameEN)
	assert.Nil(t, p.MiddleNameTH, `"" becomes nil`)
	assert.Nil(t, p.MiddleNameEN, "null becomes nil")
	assert.Nil(t, p.PassportID)
	assert.Equal(t, "1988-03-14", p.DateOfBirth.String())
	assert.Equal(t, "M", *p.Gender, "gender is upper-cased")
}

func TestSearchByID_EscapesID(t *testing.T) {
	var gotRawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, _ = NewHTTPClient(time.Second).SearchByID(context.Background(), srv.URL, "../admin?x=1")
	assert.Equal(t, "/patient/search/..%2Fadmin%3Fx=1", gotRawPath)
}

func TestSearchByID_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{"404", http.StatusNotFound, `{"error":"not found"}`, ErrNotFound},
		{"500", http.StatusInternalServerError, `oops`, ErrUnavailable},
		{"malformed JSON", http.StatusOK, `{jason`, ErrUnavailable},
		{"missing patient_hn", http.StatusOK, `{"national_id":"1100500077777"}`, ErrUnavailable},
		{"missing both IDs", http.StatusOK, `{"patient_hn":"HN1"}`, ErrUnavailable},
		{"bad date", http.StatusOK, `{"patient_hn":"HN1","national_id":"1","date_of_birth":"14/03/1988"}`, ErrUnavailable},
		{"bad gender", http.StatusOK, `{"patient_hn":"HN1","national_id":"1","gender":"X"}`, ErrUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newHIS(t, tt.status, tt.body)
			p, err := NewHTTPClient(time.Second).SearchByID(context.Background(), srv.URL, "1100500077777")
			assert.Nil(t, p)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSearchByID_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	_, err := NewHTTPClient(50*time.Millisecond).SearchByID(context.Background(), srv.URL, "1")
	assert.ErrorIs(t, err, ErrUnavailable)
}

func TestSearchByID_Unreachable(t *testing.T) {
	_, err := NewHTTPClient(time.Second).SearchByID(context.Background(), "http://127.0.0.1:1", "1")
	assert.ErrorIs(t, err, ErrUnavailable)
}
