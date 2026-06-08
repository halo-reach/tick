package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error AppError `json:"error"`
}

type DataResponse struct {
	Data any `json:"data"`
}

var (
	ErrUnauthorized    = AppError{Code: "UNAUTHORIZED", Message: "Missing or invalid API Key"}
	ErrForbidden       = AppError{Code: "FORBIDDEN", Message: "Access denied"}
	ErrNotFound        = AppError{Code: "NOT_FOUND", Message: "Resource not found"}
	ErrRateLimited     = AppError{Code: "RATE_LIMITED", Message: "Request rate exceeded"}
	ErrQuotaExceeded   = AppError{Code: "QUOTA_EXCEEDED", Message: "Quota exceeded"}
	ErrValidation      = AppError{Code: "VALIDATION_ERROR", Message: "Invalid request parameters"}
	ErrConflict        = AppError{Code: "CONFLICT", Message: "Resource conflict"}
	ErrInternal        = AppError{Code: "INTERNAL_ERROR", Message: "Internal server error"}
)

func RespondError(c *gin.Context, status int, err AppError) {
	c.JSON(status, ErrorResponse{Error: err})
}

func RespondErrorMsg(c *gin.Context, status int, code, msg string) {
	c.JSON(status, ErrorResponse{Error: AppError{Code: code, Message: msg}})
}

func RespondData(c *gin.Context, status int, data any) {
	c.JSON(status, DataResponse{Data: data})
}

func RespondRaw(c *gin.Context, status int, v any) {
	c.JSON(status, v)
}

// MarshalTruncated marshals v to JSON and truncates to maxBytes.
func MarshalTruncated(v any, maxBytes int) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	if len(b) > maxBytes {
		b = b[:maxBytes]
	}
	return string(b)
}

func StatusForError(err AppError) int {
	switch err.Code {
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "FORBIDDEN", "QUOTA_EXCEEDED":
		return http.StatusForbidden
	case "NOT_FOUND":
		return http.StatusNotFound
	case "RATE_LIMITED":
		return http.StatusTooManyRequests
	case "VALIDATION_ERROR":
		return http.StatusBadRequest
	case "CONFLICT":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
