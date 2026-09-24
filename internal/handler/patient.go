package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/middleware"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/service"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 100
)

type PatientService interface {
	Search(ctx context.Context, hospitalID int64, f model.PatientFilter) (*service.SearchResult, error)
}

type PatientHandler struct {
	svc PatientService
}

func NewPatientHandler(svc PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

type searchQuery struct {
	NationalID  string `form:"national_id" binding:"omitempty,max=13"`
	PassportID  string `form:"passport_id" binding:"omitempty,max=20"`
	FirstName   string `form:"first_name" binding:"omitempty,max=100"`
	MiddleName  string `form:"middle_name" binding:"omitempty,max=100"`
	LastName    string `form:"last_name" binding:"omitempty,max=100"`
	DateOfBirth string `form:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	PhoneNumber string `form:"phone_number" binding:"omitempty,max=20"`
	Email       string `form:"email" binding:"omitempty,email,max=255"`
	Limit       int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset      int    `form:"offset" binding:"omitempty,min=0"`
}

type searchResponse struct {
	Data   []model.Patient `json:"data"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func (h *PatientHandler) Search(c *gin.Context) {
	var q searchQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		respondError(c, apperr.Validation(validationMessage(err)))
		return
	}

	f, err := q.toFilter()
	if err != nil {
		respondError(c, err)
		return
	}

	res, err := h.svc.Search(c.Request.Context(), middleware.HospitalID(c), f)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, searchResponse{Data: res.Patients, Total: res.Total, Limit: f.Limit, Offset: f.Offset})
}

func (q searchQuery) toFilter() (model.PatientFilter, error) {
	f := model.PatientFilter{
		NationalID:  strings.TrimSpace(q.NationalID),
		PassportID:  strings.TrimSpace(q.PassportID),
		FirstName:   strings.TrimSpace(q.FirstName),
		MiddleName:  strings.TrimSpace(q.MiddleName),
		LastName:    strings.TrimSpace(q.LastName),
		PhoneNumber: strings.TrimSpace(q.PhoneNumber),
		Email:       strings.TrimSpace(q.Email),
		Limit:       q.Limit,
		Offset:      q.Offset,
	}
	if f.Limit == 0 {
		f.Limit = defaultSearchLimit
	}

	if q.DateOfBirth != "" {
		d, err := model.ParseDate(q.DateOfBirth)
		if err != nil {
			return f, apperr.Validation("date_of_birth must be in YYYY-MM-DD format")
		}
		f.DateOfBirth = &d
	}

	return f, nil
}
