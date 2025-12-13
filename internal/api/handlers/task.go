package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/task"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/utils"
)

type TaskHandler struct {
	service *services.TaskService
}

func NewTaskHandler(service *services.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

type CreateTaskRequest struct {
	Name       string                 `json:"name" binding:"required"`
	SourcePath string                 `json:"source_path" binding:"required"`
	RemoteName string                 `json:"remote_name" binding:"required"`
	RemotePath string                 `json:"remote_path" binding:"required"`
	Direction  string                 `json:"direction" binding:"required,oneof=upload download bidirectional"`
	Schedule   string                 `json:"schedule"`
	Realtime   bool                   `json:"realtime"`
	Options    map[string]interface{} `json:"options"`
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	// Validate cron schedule if provided
	if err := utils.ValidateCronSchedule(req.Schedule); err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid cron schedule format", err.Error()))
		return
	}

	task, err := h.service.CreateTask(c.Request.Context(), req.Name, req.SourcePath, req.RemoteName, req.RemotePath, req.Direction, req.Schedule, req.Realtime, req.Options)
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
				c.Error(err)
			}
		}
	}

	// If schedule is set, add to scheduler
	if req.Schedule != "" {
		scheduler, err := context.GetScheduler(c)
		if err == nil {
			if err := scheduler.AddTask(task); err != nil {
				c.Error(err)
			}
		}
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	remoteName := c.Query("remote_name")
	var tasks []*ent.Task
	var err error

	if remoteName != "" {
		tasks, err = h.service.ListTasksByRemote(c.Request.Context(), remoteName)
	} else {
		tasks, err = h.service.ListAllTasks(c.Request.Context())
	}

	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) Run(c *gin.Context) {
	idParam := c.Param("id")
	taskID, err := uuid.Parse(idParam)
	if err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid task ID", err.Error()))
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	// Retrieve TaskRunner from context
	taskRunner, err := context.GetTaskRunner(c)
	if err != nil {
		HandleError(c, NewError(http.StatusInternalServerError, err.Error(), ""))
		return
	}

	taskRunner.StartTask(task, "manual")

	c.JSON(http.StatusOK, gin.H{"message": "Task started", "task_id": task.ID.String()})
}

func (h *TaskHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid ID format", err.Error()))
		return
	}

	task, err := h.service.GetTaskWithJobs(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	c.JSON(http.StatusOK, task)
}

type UpdateTaskRequest struct {
	Name       *string                 `json:"name"`
	SourcePath *string                 `json:"source_path"`
	RemoteName *string                 `json:"remote_name"`
	RemotePath *string                 `json:"remote_path"`
	Direction  *string                 `json:"direction"`
	Schedule   *string                 `json:"schedule"`
	Realtime   *bool                   `json:"realtime"`
	Options    *map[string]interface{} `json:"options"`
}

func (h *TaskHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid ID format", err.Error()))
		return
	}

	// Get existing task
	existingTask, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		HandleError(c, NewError(http.StatusBadRequest, "Invalid request body", err.Error()))
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

	remoteName := existingTask.RemoteName
	if req.RemoteName != nil {
		remoteName = *req.RemoteName
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
			HandleError(c, NewError(http.StatusBadRequest, "Invalid cron schedule format", err.Error()))
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

	_, err = h.service.UpdateTask(c.Request.Context(), id, name, sourcePath, remoteName, remotePath, string(direction), schedule, realtime, options)
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
					c.Error(err)
				}
			} else {
				// Realtime was disabled, remove from watcher
				if err := watcher.RemoveTask(updatedTask); err != nil {
					c.Error(err)
				}
			}
		} else if realtime && existingTask.SourcePath != updatedTask.SourcePath {
			// Realtime is still enabled but source path changed, update watcher
			// Remove old path and add new path
			if err := watcher.RemoveTask(existingTask); err != nil {
				c.Error(err)
			}
			if err := watcher.AddTask(updatedTask); err != nil {
				c.Error(err)
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
					c.Error(err)
				}
			} else {
				// Schedule was removed, remove from scheduler
				if err := scheduler.RemoveTask(updatedTask); err != nil {
					c.Error(err)
				}
			}
		}
		// Note: If schedule expression unchanged, no need to update scheduler.
		// The scheduler will reload the latest task config from DB when the job runs.
	}

	c.JSON(http.StatusOK, updatedTask)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Get task before deleting to check if it has realtime enabled
	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	// If realtime is enabled, remove from watcher first
	if task.Realtime {
		watcher, err := context.GetWatcher(c)
		if err == nil {
			if err := watcher.RemoveTask(task); err != nil {
				c.Error(err)
			}
		}
	}

	// If schedule is set, remove from scheduler
	if task.Schedule != "" {
		scheduler, err := context.GetScheduler(c)
		if err == nil {
			if err := scheduler.RemoveTask(task); err != nil {
				c.Error(err)
			}
		}
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
