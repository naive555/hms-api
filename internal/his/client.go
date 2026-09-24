package his

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/naive555/hms-api/internal/model"
)

var (
	ErrNotFound    = errors.New("his: patient not found")
	ErrUnavailable = errors.New("his: unavailable")
)

const maxResponseBytes = 1 << 20 // 1 MiB

type Client interface {
	SearchByID(ctx context.Context, baseURL, id string) (*model.Patient, error)
}

type HTTPClient struct {
	http *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{http: &http.Client{Timeout: timeout}}
}

func (c *HTTPClient) SearchByID(ctx context.Context, baseURL, id string) (*model.Patient, error) {
	// {baseURL}/patient/search/{id}.
	endpoint := strings.TrimRight(baseURL, "/") + "/patient/search/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w: unexpected status %d", ErrUnavailable, resp.StatusCode)
	}

	var body patientResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&body); err != nil {
		return nil, fmt.Errorf("%w: decode response: %v", ErrUnavailable, err)
	}

	p, err := body.toModel()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid patient: %v", ErrUnavailable, err)
	}

	return p, nil
}

type patientResponse struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

func (r patientResponse) toModel() (*model.Patient, error) {
	p := &model.Patient{
		PatientHN:    strings.TrimSpace(r.PatientHN),
		FirstNameTH:  optional(r.FirstNameTH),
		MiddleNameTH: optional(r.MiddleNameTH),
		LastNameTH:   optional(r.LastNameTH),
		FirstNameEN:  optional(r.FirstNameEN),
		MiddleNameEN: optional(r.MiddleNameEN),
		LastNameEN:   optional(r.LastNameEN),
		NationalID:   optional(r.NationalID),
		PassportID:   optional(r.PassportID),
		PhoneNumber:  optional(r.PhoneNumber),
		Email:        optional(r.Email),
	}

	if p.PatientHN == "" {
		return nil, errors.New("patient_hn is required")
	}
	if p.NationalID == nil && p.PassportID == nil {
		return nil, errors.New("national_id or passport_id is required")
	}

	if s := strings.TrimSpace(r.DateOfBirth); s != "" {
		d, err := model.ParseDate(s)
		if err != nil {
			return nil, fmt.Errorf("date_of_birth %q: %w", s, err)
		}
		p.DateOfBirth = &d
	}

	if g := optional(strings.ToUpper(r.Gender)); g != nil {
		if *g != "M" && *g != "F" {
			return nil, fmt.Errorf("gender %q must be M or F", *g)
		}
		p.Gender = g
	}

	return p, nil
}

func optional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	return &s
}
