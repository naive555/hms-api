package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/naive555/hms-api/internal/apperr"
)

var registerTagNames sync.Once

func NewRouter(staffH *StaffHandler, patientH *PatientHandler, authMW gin.HandlerFunc) *gin.Engine {
	registerTagNames.Do(useJSONFieldNames)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	staff := r.Group("/staff")
	staff.POST("/create", staffH.Create)
	staff.POST("/login", staffH.Login)

	patient := r.Group("/patient", authMW)
	patient.GET("/search", patientH.Search)

	return r
}

func useJSONFieldNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, key := range []string{"json", "form"} {
			if name, _, _ := strings.Cut(f.Tag.Get(key), ","); name != "" && name != "-" {
				return name
			}
		}
		return f.Name
	})
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
		return "malformed request"
	}

	fe := ve[0]

	// the json/form tag name, see init
	field := fe.Field()

	unit := " characters"
	if fe.Kind() == reflect.Int {
		unit = ""
	}
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "min":
		return fmt.Sprintf("%s must be at least %s%s", field, fe.Param(), unit)
	case "max":
		return fmt.Sprintf("%s must be at most %s%s", field, fe.Param(), unit)
	case "email":
		return field + " must be a valid email address"
	case "datetime":
		return field + " must be in YYYY-MM-DD format"
	}

	return field + " is invalid"
}
