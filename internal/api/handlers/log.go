package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
)

type LogHandler struct {
}

// ListLogs returns job logs with flexible filtering
// Required: remote_name (enforced by connection context)
// Optional: task_id, job_id, level
func ListLogs(c *gin.Context) {
	service, err := context.GetJobService(c)
	if err != nil {
		HandleError(c, NewError(http.StatusInternalServerError, err.Error(), ""))
		return
	}

	// remote_name is required
	remoteName := c.Query("remote_name")
	if remoteName == "" {
		HandleError(c, NewError(http.StatusBadRequest, "remote_name is required", ""))
		return
	}

	// Parse optional filters
	var taskID *uuid.UUID
	if t := c.Query("task_id"); t != "" {
		parsed, err := uuid.Parse(t)
		if err == nil {
			taskID = &parsed
		}
	}

	var jobID *uuid.UUID
	if j := c.Query("job_id"); j != "" {
		parsed, err := uuid.Parse(j)
		if err == nil {
			jobID = &parsed
		}
	}

	level := c.Query("level")

	// Parse pagination
	limit := 100
	offset := 0
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	total, err := service.CountJobLogs(c.Request.Context(), remoteName, taskID, jobID, level)
	if err != nil {
		HandleError(c, err)
		return
	}

	logs, err := service.ListJobLogs(c.Request.Context(), remoteName, taskID, jobID, level, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, Page[[]*ent.JobLog]{
		Data:  logs,
		Total: total,
	})
}
