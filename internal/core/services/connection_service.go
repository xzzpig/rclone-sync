// Package services provides business logic services for the application.
package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rclone/rclone/fs/fspath"
	"github.com/xzzpig/rclone-sync/internal/core/crypto"
	"github.com/xzzpig/rclone-sync/internal/core/ent"
	"github.com/xzzpig/rclone-sync/internal/core/ent/connection"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
	"github.com/xzzpig/rclone-sync/internal/core/ports"
)

const (
	errNameEmpty          = errs.ConstError("name cannot be empty")
	errTypeEmpty          = errs.ConstError("type cannot be empty")
	errConnectionNotFound = errs.ConstError("connection not found")
)

// ConnectionService 处理云存储连接的业务逻辑
type ConnectionService struct {
	client    *ent.Client
	encryptor *crypto.Encryptor
}

// NewConnectionService 创建新的 ConnectionService 实例
func NewConnectionService(client *ent.Client, encryptor *crypto.Encryptor) *ConnectionService {
	return &ConnectionService{
		client:    client,
		encryptor: encryptor,
	}
}

// ValidateConnectionName 验证连接名称
// 使用 rclone 官方的 fspath.CheckConfigName 验证规则
func ValidateConnectionName(name string) error {
	if name == "" {
		return errNameEmpty
	}
	// 使用 rclone 官方的验证函数
	if err := fspath.CheckConfigName(name); err != nil {
		return fmt.Errorf("invalid name format: %w", err)
	}
	return nil
}

// CreateConnection 创建新的云存储连接
func (s *ConnectionService) CreateConnection(ctx context.Context, name, connType string, config map[string]string) (*ent.Connection, error) {
	// 验证名称
	if err := ValidateConnectionName(name); err != nil {
		return nil, err
	}

	// 验证类型
	if connType == "" {
		return nil, errTypeEmpty
	}

	// 检查名称是否已存在
	exists, err := s.client.Connection.
		Query().
		Where(connection.Name(name)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check connection existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("connection with name '%s' already exists", name) //nolint:err113
	}

	if config == nil {
		config = make(map[string]string)
	}
	config["type"] = connType

	// 加密配置
	encryptedConfig, err := s.encryptor.EncryptConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt config: %w", err)
	}

	// 创建连接
	conn, err := s.client.Connection.
		Create().
		SetName(name).
		SetType(connType).
		SetEncryptedConfig(encryptedConfig).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return conn, nil
}

// GetConnectionByName 根据名称获取连接
func (s *ConnectionService) GetConnectionByName(ctx context.Context, name string) (*ent.Connection, error) {
	conn, err := s.client.Connection.
		Query().
		Where(connection.Name(name)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("connection '%s' not found", name) //nolint:err113
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return conn, nil
}

// GetConnectionByID 根据 ID 获取连接
func (s *ConnectionService) GetConnectionByID(ctx context.Context, id uuid.UUID) (*ent.Connection, error) {
	conn, err := s.client.Connection.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errConnectionNotFound
		}
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	return conn, nil
}

// ListConnections 列出所有连接
func (s *ConnectionService) ListConnections(ctx context.Context) ([]*ent.Connection, error) {
	conns, err := s.client.Connection.
		Query().
		Order(ent.Asc(connection.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	return conns, nil
}

// ListConnectionsPaginated 分页列出连接
func (s *ConnectionService) ListConnectionsPaginated(ctx context.Context, limit, offset int) ([]*ent.Connection, int, error) {
	query := s.client.Connection.Query().
		Order(ent.Desc(connection.FieldCreatedAt))

	// Get total count
	totalCount, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count connections: %w", err)
	}

	// Apply pagination and fetch items
	conns, err := query.
		Limit(limit).
		Offset(offset).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list connections: %w", err)
	}

	return conns, totalCount, nil
}

// CountAssociatedTasks 返回连接关联的任务数量
func (s *ConnectionService) CountAssociatedTasks(ctx context.Context, connectionID uuid.UUID) (int, error) {
	conn, err := s.client.Connection.Get(ctx, connectionID)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, errConnectionNotFound
		}
		return 0, fmt.Errorf("failed to get connection: %w", err)
	}

	count, err := conn.QueryTasks().Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count, nil
}

// ListConnectionNames 仅返回连接名称列表（优化查询，不加载 encrypted_config）
func (s *ConnectionService) ListConnectionNames(ctx context.Context) ([]string, error) {
	conns, err := s.client.Connection.
		Query().
		Select(connection.FieldName).
		Order(ent.Asc(connection.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list connection names: %w", err)
	}

	names := make([]string, len(conns))
	for i, c := range conns {
		names[i] = c.Name
	}
	return names, nil
}

// GetConnectionConfig 获取连接的解密配置（用于编辑）- 按名称
func (s *ConnectionService) GetConnectionConfig(ctx context.Context, name string) (map[string]string, error) {
	conn, err := s.GetConnectionByName(ctx, name)
	if err != nil {
		return nil, err
	}

	config, err := s.encryptor.DecryptConfig(conn.EncryptedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}
	config["type"] = conn.Type

	return config, nil
}

// GetConnectionConfigByID 获取连接的解密配置（用于编辑）- 按 ID
func (s *ConnectionService) GetConnectionConfigByID(ctx context.Context, id uuid.UUID) (map[string]string, error) {
	conn, err := s.GetConnectionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	config, err := s.encryptor.DecryptConfig(conn.EncryptedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	return config, nil
}

// UpdateConnection 更新连接配置（基于 ID）
func (s *ConnectionService) UpdateConnection(ctx context.Context, id uuid.UUID, name, connType *string, config map[string]string) error {
	// 根据 ID 查询连接
	conn, err := s.client.Connection.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return errConnectionNotFound
		}
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// 构建更新查询
	update := s.client.Connection.UpdateOne(conn)

	// 如果提供了新名称，验证并更新
	if name != nil && *name != conn.Name {
		if err := ValidateConnectionName(*name); err != nil {
			return err
		}
		// 检查新名称是否已存在
		exists, err := s.client.Connection.
			Query().
			Where(connection.Name(*name)).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("failed to check new name existence: %w", err)
		}
		if exists {
			return fmt.Errorf("connection with name '%s' already exists", *name) //nolint:err113
		}
		update = update.SetName(*name)
	}

	// 如果提供了新类型，更新
	if connType != nil {
		update = update.SetType(*connType)
	}

	if config == nil {
		config = make(map[string]string)
	}
	config["type"] = conn.Type

	// 加密并更新配置（config 是必需的）
	encryptedConfig, err := s.encryptor.EncryptConfig(config)
	if err != nil {
		return fmt.Errorf("failed to encrypt config: %w", err)
	}
	update = update.SetEncryptedConfig(encryptedConfig)

	// 保存更新
	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update connection: %w", err)
	}

	return nil
}

// DeleteConnectionByName 根据名称删除连接（级联删除关联的任务）
func (s *ConnectionService) DeleteConnectionByName(ctx context.Context, name string) error {
	conn, err := s.GetConnectionByName(ctx, name)
	if err != nil {
		return err
	}

	// Ent 会自动处理级联删除（schema 中已定义）
	err = s.client.Connection.DeleteOne(conn).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

// DeleteConnectionByID 根据 ID 删除连接（级联删除关联的任务）
func (s *ConnectionService) DeleteConnectionByID(ctx context.Context, id uuid.UUID) error {
	conn, err := s.GetConnectionByID(ctx, id)
	if err != nil {
		return err
	}

	// Ent 会自动处理级联删除（schema 中已定义）
	err = s.client.Connection.DeleteOne(conn).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	return nil
}

// HasAssociatedTasks 检查连接是否有关联的任务
func (s *ConnectionService) HasAssociatedTasks(ctx context.Context, connectionID uuid.UUID) (bool, error) {
	conn, err := s.client.Connection.Get(ctx, connectionID)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, errConnectionNotFound
		}
		return false, fmt.Errorf("failed to get connection: %w", err)
	}

	count, err := conn.QueryTasks().Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count > 0, nil
}

var _ ports.ConnectionService = (*ConnectionService)(nil)
