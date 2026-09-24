package middleware

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naive555/hms-api/internal/apperr"
	"github.com/naive555/hms-api/internal/auth"
)

const (
	ctxStaffID    = "auth.staff_id"
	ctxHospitalID = "auth.hospital_id"
)

// Auth verifies the Bearer token and stores the staff and hospital IDs in the
// request context.
func Auth(tokens auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		trimmedTok := strings.TrimSpace(token)
		if !ok || !strings.EqualFold(scheme, "Bearer") || trimmedTok == "" {
			abort(c, "missing or malformed Authorization header")
			return
		}

		claims, err := tokens.Parse(trimmedTok)
		if err != nil {
			abort(c, err.Error())
			return
		}

		staffID, err := claims.StaffID()
		if err != nil || staffID <= 0 || claims.HospitalID <= 0 {
			abort(c, "token has invalid subject or hospital_id")
			return
		}

		c.Set(ctxStaffID, staffID)
		c.Set(ctxHospitalID, claims.HospitalID)
		c.Next()
	}
}

// Returns the authenticated hospital's ID.
func HospitalID(c *gin.Context) int64 { return c.GetInt64(ctxHospitalID) }

// Returns the authenticated staff member's ID.
func StaffID(c *gin.Context) int64 { return c.GetInt64(ctxStaffID) }

// Every failure returns the same 401.
func abort(c *gin.Context, reason string) {
	slog.DebugContext(c.Request.Context(), "auth rejected", "reason", reason, "path", c.FullPath())
	c.AbortWithStatusJSON(apperr.ErrUnauthorized.Status, apperr.ErrUnauthorized.Body())
}
