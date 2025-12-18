package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// ImportHandler handles import-related HTTP requests.
type ImportHandler struct {
	connService ports.ConnectionService
}

// NewImportHandler creates a new import handler
func NewImportHandler(connService ports.ConnectionService) *ImportHandler {
	return &ImportHandler{
		connService: connService,
	}
}

// ParseRequest represents the request body for parsing rclone.conf
type ParseRequest struct {
	Content string `json:"content"`
}

// ParseResponse represents the response for parsing rclone.conf
type ParseResponse struct {
	Connections []rclone.ParsedConnection `json:"connections"`
	Validation  *rclone.ValidationResult  `json:"validation,omitempty"`
}

// ExecuteRequest represents the request body for executing import
type ExecuteRequest struct {
	Connections []ConnectionToImport `json:"connections" binding:"required"`
	Overwrite   bool                 `json:"overwrite"`
}

// ConnectionToImport represents a connection to import
type ConnectionToImport struct {
	Name   string            `json:"name" binding:"required"`
	Type   string            `json:"type" binding:"required"`
	Config map[string]string `json:"config" binding:"required"`
}

// ExecuteResponse represents the response for executing import
type ExecuteResponse struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// Parse handles POST /import/parse
// @Summary Parse rclone.conf content
// @Description Parse rclone.conf content and validate against existing connections
// @Tags import
// @Accept json
// @Produce json
// @Param request body ParseRequest true "rclone.conf content"
// @Success 200 {object} ParseResponse
// @Failure 400 {object} ErrorResponse
// @Router /import/parse [post]
func (h *ImportHandler) Parse(c *gin.Context) {
	ctx := c.Request.Context()

	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}

	// Parse rclone.conf content
	connections, err := rclone.ParseRcloneConf(req.Content)
	if err != nil {
		appErr := NewLocalizedError(c, http.StatusBadRequest, i18n.ErrImportParseFailed, err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}

	// Get existing connection names for validation
	existingConns, err := h.connService.ListConnections(ctx)
	if err != nil {
		appErr := NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrDatabaseError, err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}

	existingNames := make([]string, len(existingConns))
	for i, conn := range existingConns {
		existingNames[i] = conn.Name
	}

	// Validate import
	validation := rclone.ValidateImport(connections, existingNames)

	c.JSON(http.StatusOK, ParseResponse{
		Connections: connections,
		Validation:  validation,
	})
}

// Execute handles POST /import/execute
// @Summary Execute import of connections
// @Description Import connections into the database, optionally overwriting existing ones
// @Tags import
// @Accept json
// @Produce json
// @Param request body ExecuteRequest true "connections to import"
// @Success 200 {object} ExecuteResponse
// @Failure 400 {object} ErrorResponse
// @Router /import/execute [post]
func (h *ImportHandler) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appErr := NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error())
		c.JSON(appErr.Code, appErr)
		return
	}

	if len(req.Connections) == 0 {
		appErr := NewLocalizedError(c, http.StatusBadRequest, i18n.ErrImportEmptyList, "")
		c.JSON(appErr.Code, appErr)
		return
	}

	var imported, skipped, failed int
	var errs []string

	for _, conn := range req.Connections {
		err := h.importConnection(ctx, conn, req.Overwrite)
		if err != nil {
			if errors.Is(err, ErrConnectionAlreadyExists) && !req.Overwrite {
				skipped++
			} else {
				failed++
				errs = append(errs, err.Error())
			}
		} else {
			imported++
		}
	}

	c.JSON(http.StatusOK, ExecuteResponse{
		Imported: imported,
		Skipped:  skipped,
		Failed:   failed,
		Errors:   errs,
	})
}

// importConnection imports a single connection
func (h *ImportHandler) importConnection(ctx context.Context, conn ConnectionToImport, overwrite bool) error {
	// Check if connection already exists
	existing, err := h.connService.GetConnectionByName(ctx, conn.Name)
	if err == nil && existing != nil {
		// Connection exists
		if !overwrite {
			return ErrConnectionAlreadyExists
		}

		// Overwrite: update existing connection
		// UpdateConnection requires UUID and pointer parameters
		namePtr := &conn.Name
		typePtr := &conn.Type
		return h.connService.UpdateConnection(ctx, existing.ID, namePtr, typePtr, conn.Config)
	}

	// Create new connection
	_, err = h.connService.CreateConnection(ctx, conn.Name, conn.Type, conn.Config)
	return err
}

// ErrConnectionAlreadyExists is returned when a connection already exists
const ErrConnectionAlreadyExists = errs.ConstError("connection already exists")
