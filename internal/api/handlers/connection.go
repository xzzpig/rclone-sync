// Package handlers provides HTTP request handlers for the API.
package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/services"
	"github.com/xzzpig/rclone-sync/internal/rclone"
)

// ConnectionHandler 处理连接相关的 HTTP 请求
type ConnectionHandler struct {
	connService *services.ConnectionService
}

// NewConnectionHandler 创建新的 ConnectionHandler 实例
func NewConnectionHandler(connService *services.ConnectionService) *ConnectionHandler {
	return &ConnectionHandler{
		connService: connService,
	}
}

// ConnectionRequest 创建/更新连接的请求体
type ConnectionRequest struct {
	Name   string            `json:"name" binding:"required"`
	Type   string            `json:"type" binding:"required"`
	Config map[string]string `json:"config" binding:"required"`
}

// ConnectionResponse 连接响应体（不包含加密配置）
type ConnectionResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// toConnectionResponse 将 ent.Connection 转换为 ConnectionResponse
func toConnectionResponse(conn *ent.Connection) ConnectionResponse {
	return ConnectionResponse{
		ID:        conn.ID.String(),
		Name:      conn.Name,
		Type:      conn.Type,
		CreatedAt: conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Create 处理 POST /connections 请求
// @Summary 创建新的云存储连接
// @Tags connections
// @Accept json
// @Produce json
// @Param connection body ConnectionRequest true "连接信息"
// @Success 201 {object} ConnectionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections [post]
func (h *ConnectionHandler) Create(c *gin.Context) {
	var req ConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 验证必填字段
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}
	if req.Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config is required"})
		return
	}

	// 创建连接
	conn, err := h.connService.CreateConnection(c.Request.Context(), req.Name, req.Type, req.Config)
	if err != nil {
		// 检查是否是名称重复错误
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		// 检查是否是验证错误
		if strings.Contains(err.Error(), "name") || strings.Contains(err.Error(), "format") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toConnectionResponse(conn))
}

// List 处理 GET /connections 请求
// @Summary 列出所有连接
// @Tags connections
// @Produce json
// @Success 200 {array} ConnectionResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections [get]
func (h *ConnectionHandler) List(c *gin.Context) {
	conns, err := h.connService.ListConnections(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list connections: " + err.Error()})
		return
	}

	// 转换为响应格式
	responses := make([]ConnectionResponse, len(conns))
	for i, conn := range conns {
		responses[i] = toConnectionResponse(conn)
	}

	c.JSON(http.StatusOK, responses)
}

// Get 处理 GET /connections/:id 请求
// @Summary 获取连接详情
// @Tags connections
// @Produce json
// @Param id path string true "连接ID"
// @Success 200 {object} ConnectionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id} [get]
func (h *ConnectionHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	conn, err := h.connService.GetConnectionByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connection: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, toConnectionResponse(conn))
}

// GetConfig 处理 GET /connections/:id/config 请求
// @Summary 获取连接配置（解密后）
// @Tags connections
// @Produce json
// @Param id path string true "连接ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id}/config [get]
func (h *ConnectionHandler) GetConfig(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	config, err := h.connService.GetConnectionConfigByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connection config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateRequest 更新连接的请求体
type UpdateRequest struct {
	Name   string            `json:"name"`   // 可选，新名称
	Type   string            `json:"type"`   // 可选，新类型
	Config map[string]string `json:"config"` // 可选，新配置
}

// Update 处理 PUT /connections/:id 请求
// @Summary 更新连接
// @Tags connections
// @Accept json
// @Produce json
// @Param id path string true "连接ID"
// @Param connection body UpdateRequest true "更新信息"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id} [put]
func (h *ConnectionHandler) Update(c *gin.Context) {
	// 解析 ID 参数
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 验证必填字段 - config 是必需的
	if req.Config == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config is required"})
		return
	}

	// 准备可选参数
	var namePtr, typePtr *string
	if req.Name != "" {
		namePtr = &req.Name
	}
	if req.Type != "" {
		typePtr = &req.Type
	}

	// 更新连接
	err = h.connService.UpdateConnection(c.Request.Context(), id, namePtr, typePtr, req.Config)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "name") || strings.Contains(err.Error(), "format") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update connection: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestConfigRequest 测试未保存配置的请求体
type TestConfigRequest struct {
	Type   string            `json:"type" binding:"required"`
	Config map[string]string `json:"config" binding:"required"`
}

// TestUnsavedConfig 处理 POST /connections/test 请求
// @Summary 测试未保存的连接配置
// @Tags connections
// @Accept json
// @Produce json
// @Param config body TestConfigRequest true "配置信息"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/test [post]
func (h *ConnectionHandler) TestUnsavedConfig(c *gin.Context) {
	var req TestConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	// Test the remote configuration using rclone's TestRemote functionality
	if err := rclone.TestRemote(c.Request.Context(), req.Type, req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Test 处理 POST /connections/:id/test 请求
// @Summary 测试已保存的连接
// @Tags connections
// @Produce json
// @Param id path string true "连接ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id}/test [post]
func (h *ConnectionHandler) Test(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	// Get connection to get the type
	conn, err := h.connService.GetConnectionByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connection: " + err.Error()})
		return
	}

	// Get connection config
	config, err := h.connService.GetConnectionConfigByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connection config: " + err.Error()})
		return
	}

	// Test the remote configuration
	if err := rclone.TestRemote(c.Request.Context(), conn.Type, config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// QuotaResponse 配额信息响应
type QuotaResponse struct {
	Total   *int64 `json:"total,omitempty"`
	Used    *int64 `json:"used,omitempty"`
	Free    *int64 `json:"free,omitempty"`
	Trashed *int64 `json:"trashed,omitempty"`
	Other   *int64 `json:"other,omitempty"`
}

// GetQuota 处理 GET /connections/:id/quota 请求
// @Summary 获取连接的配额信息
// @Tags connections
// @Produce json
// @Param id path string true "连接ID"
// @Success 200 {object} QuotaResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id}/quota [get]
func (h *ConnectionHandler) GetQuota(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	// Verify connection exists and get its name for rclone
	conn, err := h.connService.GetConnectionByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get connection: " + err.Error()})
		return
	}

	// Get quota from rclone (using connection name for rclone compatibility)
	quota, err := rclone.GetRemoteQuota(c.Request.Context(), conn.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get quota: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, &QuotaResponse{
		Total:   quota.Total,
		Used:    quota.Used,
		Free:    quota.Free,
		Trashed: quota.Trashed,
		Other:   quota.Other,
	})
}

// Delete 处理 DELETE /connections/:id 请求
// @Summary 删除连接（级联删除关联的任务）
// @Tags connections
// @Param id path string true "连接ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /connections/{id} [delete]
func (h *ConnectionHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id parameter is required"})
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format: " + err.Error()})
		return
	}

	// 删除连接（数据库会自动级联删除关联的任务）
	err = h.connService.DeleteConnectionByID(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete connection: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
