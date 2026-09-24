package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/naive555/hms-api/internal/apperr"
)

func NewRouter(staffH *StaffHandler /*, patientH, authMW in next step */) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	staff := r.Group("/staff")
	staff.POST("/create", staffH.Create)
	staff.POST("/login", staffH.Login)
	return r
}

func respondError(c *gin.Context, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		slog.ErrorContext(c.Request.Context(), "unhandled error", "error", err, "path", c.FullPath())
		ae = apperr.ErrInternal
	}
	c.AbortWithStatusJSON(ae.Status, ae.Body())
}

func validationMessage(err error) string {
	var ve validator.ValidationErrors

	if !errors.As(err, &ve) {
		return "invalid request body"
	}

	fe := ve[0]
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	}

	return field + " is invalid"
}
