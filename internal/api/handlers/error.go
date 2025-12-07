package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

// AppError represents a structured error response
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// NewError creates a new AppError
func NewError(code int, message string, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// HandleError processes errors and sends a JSON response
func HandleError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.Code, appErr)
		return
	}

	// Helper to check for domain errors
	if errors.Is(err, errs.ErrNotFound) {
		c.JSON(http.StatusNotFound, &AppError{
			Code:    http.StatusNotFound,
			Message: "Resource Not Found",
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrAlreadyExists) {
		c.JSON(http.StatusConflict, &AppError{
			Code:    http.StatusConflict,
			Message: "Resource Already Exists",
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, &AppError{
			Code:    http.StatusBadRequest,
			Message: "Invalid Input",
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, &AppError{
			Code:    http.StatusUnauthorized,
			Message: "Unauthorized",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, &AppError{
		Code:    http.StatusInternalServerError,
		Message: "Internal Server Error",
		Details: err.Error(),
	})
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(c *gin.Context) {
	c.JSON(http.StatusNotFound, &AppError{
		Code:    http.StatusNotFound,
		Message: "Resource Not Found",
	})
}
