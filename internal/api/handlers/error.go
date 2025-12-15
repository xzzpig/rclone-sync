package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	apicontext "github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/i18n"
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

// NewLocalizedError creates a new AppError with localized message
// msgKey is the i18n message key, details can be error string for debugging
func NewLocalizedError(c *gin.Context, code int, msgKey string, details string) *AppError {
	localizer := apicontext.GetLocalizer(c)
	return &AppError{
		Code:    code,
		Message: i18n.T(localizer, msgKey),
		Details: details,
	}
}

// HandleError processes errors and sends a JSON response
// Note: I18nError types are handled by I18nErrorMiddleware
func HandleError(c *gin.Context, err error) {
	// Get localizer for translations
	localizer := apicontext.GetLocalizer(c)

	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.Code, appErr)
		return
	}

	// Helper to check for domain errors
	if errors.Is(err, errs.ErrNotFound) {
		c.JSON(http.StatusNotFound, &AppError{
			Code:    http.StatusNotFound,
			Message: i18n.T(localizer, i18n.ErrNotFound),
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrAlreadyExists) {
		c.JSON(http.StatusConflict, &AppError{
			Code:    http.StatusConflict,
			Message: i18n.T(localizer, i18n.ErrAlreadyExists),
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrInvalidInput) {
		c.JSON(http.StatusBadRequest, &AppError{
			Code:    http.StatusBadRequest,
			Message: i18n.T(localizer, i18n.ErrInvalidInput),
			Details: err.Error(),
		})
		return
	}
	if errors.Is(err, errs.ErrUnauthorized) {
		c.JSON(http.StatusUnauthorized, &AppError{
			Code:    http.StatusUnauthorized,
			Message: i18n.T(localizer, i18n.ErrUnauthorized),
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, &AppError{
		Code:    http.StatusInternalServerError,
		Message: i18n.T(localizer, i18n.ErrGeneric),
		Details: err.Error(),
	})
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(c *gin.Context) {
	localizer := apicontext.GetLocalizer(c)
	c.JSON(http.StatusNotFound, &AppError{
		Code:    http.StatusNotFound,
		Message: i18n.T(localizer, i18n.ErrNotFound),
	})
}
