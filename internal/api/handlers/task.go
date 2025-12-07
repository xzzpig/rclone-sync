package handlers

import (
	"net/http"

	"github.com/xzzpig/rclone-sync/internal/api/context"
	"github.com/xzzpig/rclone-sync/internal/core/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	task, err := h.service.CreateTask(c.Request.Context(), req.Name, req.SourcePath, req.RemoteName, req.RemotePath, req.Direction, req.Schedule, req.Realtime, req.Options)
	if err != nil {
		HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	tasks, err := h.service.ListAllTasks(c.Request.Context())
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

	task, err := h.service.GetTask(c.Request.Context(), id)
	if err != nil {
		HandleError(c, NewError(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.UpdateTask(c.Request.Context(), id, req.Name, req.SourcePath, req.RemoteName, req.RemotePath, req.Direction, req.Schedule, req.Realtime, req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.service.DeleteTask(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
