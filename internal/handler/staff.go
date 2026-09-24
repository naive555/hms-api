package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/model"
	"github.com/naive555/hms-api/internal/service"
)

type AuthService interface {
	Register(ctx context.Context, in service.RegisterInput) (*model.Staff, error)
	Login(ctx context.Context, in service.LoginInput) (*service.LoginResult, error)
}

type credentialsRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Hospital string `json:"hospital" binding:"required"`
}

type StaffHandler struct {
	svc AuthService
}

func NewStaffHandler(svc AuthService) *StaffHandler {
	return &StaffHandler{
		svc: svc,
	}
}

func (h *StaffHandler) Create(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(validationMessage(err)))
		return
	}

	st, err := h.svc.Register(c.Request.Context(), service.RegisterInput{
		Username: req.Username, Password: req.Password, HospitalCode: req.Hospital,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": st.ID, "username": st.Username, "hospital": req.Hospital})
}

func (h *StaffHandler) Login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperr.Validation(validationMessage(err)))
		return
	}

	res, err := h.svc.Login(c.Request.Context(), service.LoginInput{
		Username: req.Username, Password: req.Password, HospitalCode: req.Hospital,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": res.Token,
		"token_type":   "Bearer",
		"expires_in":   int(res.ExpiresIn.Seconds()),
	})
}
