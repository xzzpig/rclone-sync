package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// ListProviders returns all available providers.
func ListProviders(c *gin.Context) {
	providers := rclone.ListProviders()
	c.JSON(http.StatusOK, providers)
}

// GetProviderOptions returns options schema for a specific provider.
func GetProviderOptions(c *gin.Context) {
	name := c.Param("name")
	schema, err := rclone.GetProviderOptions(name)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrProviderNotFound, err.Error()))
		return
	}
	c.JSON(http.StatusOK, schema)
}
