package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// ListRemotes returns all configured remotes.
func ListRemotes(c *gin.Context) {
	remotes, err := rclone.ListRemotesWithInfo()
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrFailedToListRemotes, err.Error()))
		return
	}
	c.JSON(http.StatusOK, remotes)
}

// GetRemoteInfo returns configuration for a specific remote.
func GetRemoteInfo(c *gin.Context) {
	name := c.Param("name")
	info, err := rclone.GetRemoteConfig(name)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrRemoteNotFound, err.Error()))
		return
	}
	c.JSON(http.StatusOK, info)
}

// CreateRemote creates or updates a remote.
func CreateRemote(c *gin.Context) {
	name := c.Param("name")
	var params map[string]string
	if err := c.ShouldBindJSON(&params); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error()))
		return
	}

	if err := rclone.CreateRemote(name, params); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrFailedToCreateRemote, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DeleteRemote deletes a remote.
func DeleteRemote(c *gin.Context) {
	name := c.Param("name")
	rclone.DeleteRemote(name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

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

type TestRemoteRequest struct {
	Provider string            `json:"provider" binding:"required"`
	Params   map[string]string `json:"params" binding:"required"`
}

// TestRemote verifies connection settings for a new remote.
func TestRemote(c *gin.Context) {
	var req TestRemoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error()))
		return
	}

	if err := rclone.TestRemote(c.Request.Context(), req.Provider, req.Params); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrConnectionTestFailed, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetRemoteQuota returns quota information for a specific remote.
func GetRemoteQuota(c *gin.Context) {
	name := c.Param("name")
	quota, err := rclone.GetRemoteQuota(c.Request.Context(), name)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrFailedToGetQuota, err.Error()))
		return
	}
	c.JSON(http.StatusOK, quota)
}
