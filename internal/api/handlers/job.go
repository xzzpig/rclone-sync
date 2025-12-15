package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/i18n"
)

type JobHandler struct {
}

// GetJobProgress returns the realtime progress of a running job
func GetJobProgress(c *gin.Context) {
	idParam := c.Param("id")
	jobID, err := uuid.Parse(idParam)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, err.Error()))
		return
	}

	// Retrieve SyncEngine from context
	syncEngine, err := context.GetSyncEngine(c)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrGeneric, err.Error()))
		return
	}

	progress, ok := syncEngine.GetJobProgress(jobID)
	if !ok {
		// If not running/found in memory, client should fallback to DB status
		// HTTP 404 indicates "not currently active in memory"
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrJobNotActive, ""))
		return
	}

	c.JSON(http.StatusOK, progress)
}

// ListJobs returns a list of jobs
func ListJobs(c *gin.Context) {
	service, err := context.GetJobService(c)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrGeneric, err.Error()))
		return
	}

	limit := 10
	offset := 0

	// Parse pagination
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	var taskID *uuid.UUID
	if t := c.Query("task_id"); t != "" {
		parsed, err := uuid.Parse(t)
		if err == nil {
			taskID = &parsed
		}
	}

	// Support filtering by remote_name
	remoteName := c.Query("remote_name")

	total, err := service.CountJobs(c.Request.Context(), taskID, remoteName)
	if err != nil {
		HandleError(c, err)
		return
	}

	jobs, err := service.ListJobs(c.Request.Context(), taskID, remoteName, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, Page[[]*ent.Job]{
		Data:  jobs,
		Total: total,
	})
}

// GetJob returns a single job with logs
func GetJob(c *gin.Context) {
	idParam := c.Param("id")
	jobID, err := uuid.Parse(idParam)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, err.Error()))
		return
	}

	service, err := context.GetJobService(c)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrGeneric, err.Error()))
		return
	}

	job, err := service.GetJobWithLogs(c.Request.Context(), jobID)
	if err != nil {
		if ent.IsNotFound(err) {
			HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrJobNotFound, err.Error()))
		} else {
			HandleError(c, err)
		}
		return
	}

	c.JSON(http.StatusOK, job)
}
