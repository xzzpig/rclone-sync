package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/i18n"
	"github.com/xzzpig/rclone-sync/internal/utils"
)

// TaskHandler handles task-related HTTP requests.
type TaskHandler struct {
	service *services.TaskService
}

// NewTaskHandler creates a new TaskHandler with the given service.
func NewTaskHandler(service *services.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	Name         string                 `json:"name" binding:"required"`
	SourcePath   string                 `json:"source_path" binding:"required"`
	ConnectionID string                 `json:"connection_id" binding:"required"`
	RemotePath   string                 `json:"remote_path" binding:"required"`
	Direction    string                 `json:"direction" binding:"required,oneof=upload download bidirectional"`
	Schedule     string                 `json:"schedule"`
	Realtime     bool                   `json:"realtime"`
	Options      map[string]interface{} `json:"options"`
}

// Create handles POST /tasks to create a new task.
func (h *TaskHandler) Create(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error()))
		return
	}

	// Validate cron schedule if provided
	if err := utils.ValidateCronSchedule(req.Schedule); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidSchedule, err.Error()))
		return
	}

	// Parse connection_id UUID
	connectionID, err := uuid.Parse(req.ConnectionID)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, "invalid connection_id format"))
		return
	}

	task, err := h.service.CreateTask(c.Request.Context(), req.Name, req.SourcePath, connectionID, req.RemotePath, req.Direction, req.Schedule, req.Realtime, req.Options)
	if err != nil {
		HandleError(c, err)
		return
	}

	// If realtime sync is enabled, add to watcher
	if req.Realtime {
		watcher, err := context.GetWatcher(c)
		if err == nil {
			if err := watcher.AddTask(task); err != nil {
				// Log the error but don't fail the request
				// The task was created successfully, watcher can be added later
				_ = c.Error(err)
			}
		}
	}

	// If schedule is set, add to scheduler
	if req.Schedule != "" {
		scheduler, err := context.GetScheduler(c)
		if err == nil {
			if err := scheduler.AddTask(task); err != nil {
				_ = c.Error(err)
			}
		}
	}

	c.JSON(http.StatusCreated, task)
}

// List handles GET /tasks to list all tasks.
func (h *TaskHandler) List(c *gin.Context) {
	connectionIDStr := c.Query("connection_id")
	var tasks []*ent.Task
	var err error

	if connectionIDStr != "" {
		connectionID, parseErr := uuid.Parse(connectionIDStr)
		if parseErr != nil {
			HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, "invalid connection_id format"))
			return
		}
		tasks, err = h.service.ListTasksByConnection(c.Request.Context(), connectionID)
	} else {
		tasks, err = h.service.ListAllTasks(c.Request.Context())
	}

	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// Run handles POST /tasks/:id/run to start a task execution.
func (h *TaskHandler) Run(c *gin.Context) {
	idParam := c.Param("id")
	taskID, err := uuid.Parse(idParam)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, err.Error()))
		return
	}

	task, err := h.service.GetTaskWithConnection(c.Request.Context(), taskID)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrTaskNotFound, err.Error()))
		return
	}

	// Retrieve TaskRunner from context
	taskRunner, err := context.GetTaskRunner(c)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrGeneric, err.Error()))
		return
	}

	_ = taskRunner.StartTask(task, "manual")

	c.JSON(http.StatusOK, gin.H{"message": "Task started", "task_id": task.ID.String()})
}

// Get handles GET /tasks/:id to get a single task.
func (h *TaskHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, err.Error()))
		return
	}

	task, err := h.service.GetTaskWithJobs(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrTaskNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTaskRequest represents the request body for updating a task.
type UpdateTaskRequest struct {
	Name         *string                 `json:"name"`
	SourcePath   *string                 `json:"source_path"`
	ConnectionID *string                 `json:"connection_id"` // Optional: allow changing connection
	RemotePath   *string                 `json:"remote_path"`
	Direction    *string                 `json:"direction"`
	Schedule     *string                 `json:"schedule"`
	Realtime     *bool                   `json:"realtime"`
	Options      *map[string]interface{} `json:"options"`
}

// Update handles PUT /tasks/:id to update a task.
func (h *TaskHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, err.Error()))
		return
	}

	// Get existing task
	existingTask, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrTaskNotFound, err.Error()))
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidRequestBody, err.Error()))
		return
	}

	// Use existing values if not provided in update
	name := existingTask.Name
	if req.Name != nil {
		name = *req.Name
	}

	sourcePath := existingTask.SourcePath
	if req.SourcePath != nil {
		sourcePath = *req.SourcePath
	}

	// Get existing connection_id from task
	connectionID := existingTask.ConnectionID
	if req.ConnectionID != nil {
		// Parse and validate new connection_id
		parsedID, err := uuid.Parse(*req.ConnectionID)
		if err != nil {
			HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, "invalid connection_id format"))
			return
		}
		connectionID = parsedID
	}

	remotePath := existingTask.RemotePath
	if req.RemotePath != nil {
		remotePath = *req.RemotePath
	}

	direction := existingTask.Direction
	if req.Direction != nil {
		direction = task.Direction(*req.Direction)
	}

	schedule := existingTask.Schedule
	if req.Schedule != nil {
		schedule = *req.Schedule
		// Validate cron schedule if provided
		if err := utils.ValidateCronSchedule(schedule); err != nil {
			HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidSchedule, err.Error()))
			return
		}
	}

	realtime := existingTask.Realtime
	if req.Realtime != nil {
		realtime = *req.Realtime
	}

	options := existingTask.Options
	if req.Options != nil {
		options = *req.Options
	}

	_, err = h.service.UpdateTask(c.Request.Context(), id, name, sourcePath, connectionID, remotePath, string(direction), schedule, realtime, options)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Reload task with jobs to include latest job information
	updatedTask, err := h.service.GetTaskWithJobs(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	// Handle watcher updates based on realtime status changes
	watcher, err := context.GetWatcher(c)
	if err == nil {
		// Check if realtime status changed
		if existingTask.Realtime != realtime {
			if realtime {
				// Realtime was enabled, add to watcher
				if err := watcher.AddTask(updatedTask); err != nil {
					_ = c.Error(err)
				}
			} else {
				// Realtime was disabled, remove from watcher
				if err := watcher.RemoveTask(updatedTask); err != nil {
					_ = c.Error(err)
				}
			}
		} else if realtime && existingTask.SourcePath != updatedTask.SourcePath {
			// Realtime is still enabled but source path changed, update watcher
			// Remove old path and add new path
			if err := watcher.RemoveTask(existingTask); err != nil {
				_ = c.Error(err)
			}
			if err := watcher.AddTask(updatedTask); err != nil {
				_ = c.Error(err)
			}
		}
	}

	// Handle scheduler updates based on schedule changes
	scheduler, err := context.GetScheduler(c)
	if err == nil {
		// Check if schedule changed
		if existingTask.Schedule != schedule {
			if schedule != "" {
				// Schedule was added/updated, add to scheduler
				// AddTask internally removes existing job before adding new one
				if err := scheduler.AddTask(updatedTask); err != nil {
					_ = c.Error(err)
				}
			} else {
				// Schedule was removed, remove from scheduler
				if err := scheduler.RemoveTask(updatedTask); err != nil {
					_ = c.Error(err)
				}
			}
		}
		// Note: If schedule expression unchanged, no need to update scheduler.
		// The scheduler will reload the latest task config from DB when the job runs.
	}

	c.JSON(http.StatusOK, updatedTask)
}

// Delete handles DELETE /tasks/:id to delete a task.
func (h *TaskHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusBadRequest, i18n.ErrInvalidIDFormat, ""))
		return
	}

	// Get task before deleting to check if it has realtime enabled
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusNotFound, i18n.ErrTaskNotFound, err.Error()))
		return
	}

	// If realtime is enabled, remove from watcher first
	if task.Realtime {
		watcher, err := context.GetWatcher(c)
		if err == nil {
			if err := watcher.RemoveTask(task); err != nil {
				_ = c.Error(err)
			}
		}
	}

	// If schedule is set, remove from scheduler
	if task.Schedule != "" {
		scheduler, err := context.GetScheduler(c)
		if err == nil {
			if err := scheduler.RemoveTask(task); err != nil {
				_ = c.Error(err)
			}
		}
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		HandleError(c, NewLocalizedError(c, http.StatusInternalServerError, i18n.ErrGeneric, err.Error()))
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
