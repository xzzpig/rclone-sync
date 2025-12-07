package handlers

import (
	"net/http"

	"github.com/xzzpig/rclone-sync/internal/rclone"

	"github.com/gin-gonic/gin"
)

// ListRemotes returns all configured remotes.
func ListRemotes(c *gin.Context) {
	remotes := rclone.ListRemotes()
	c.JSON(http.StatusOK, remotes)
}

// GetRemoteInfo returns configuration for a specific remote.
func GetRemoteInfo(c *gin.Context) {
	name := c.Param("name")
	info, err := rclone.GetRemoteInfo(name)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Remote not found", err.Error()))
		return
	}
	c.JSON(http.StatusOK, info)
}

// CreateRemote creates or updates a remote.
func CreateRemote(c *gin.Context) {
	name := c.Param("name")
	var params map[string]string
	if err := c.ShouldBindJSON(&params); err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	if err := rclone.CreateRemote(name, params); err != nil {
		HandleError(c, NewError(http.StatusInternalServerError, "Failed to create remote", err.Error()))
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
		HandleError(c, NewError(http.StatusNotFound, "Provider not found", err.Error()))
		return
	}
	c.JSON(http.StatusOK, schema)
}
