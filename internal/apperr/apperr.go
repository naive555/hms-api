package apperr

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

type Body struct {
	Error BodyError `json:"error"`
}

type BodyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Body() Body { return Body{BodyError{Code: e.Code, Message: e.Message}} }

func Validation(msg string) *Error { return &Error{http.StatusBadRequest, "VALIDATION_ERROR", msg} }

var (
	ErrInternal           = &Error{http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error"}
	ErrUnauthorized       = &Error{http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid token"}
	ErrInvalidCredentials = &Error{http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password"}
	ErrHospitalNotFound   = &Error{http.StatusNotFound, "HOSPITAL_NOT_FOUND", "hospital not found"}
	ErrUsernameTaken      = &Error{http.StatusConflict, "USERNAME_TAKEN", "username already exists in this hospital"}
)
