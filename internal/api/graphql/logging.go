package graphql

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/xzzpig/rclone-sync/internal/core/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggingExtension provides logging for GraphQL operations.
type LoggingExtension struct {
	Logger *zap.Logger
}

// NewLoggingExtension creates a new logging extension.
func NewLoggingExtension() *LoggingExtension {
	return &LoggingExtension{
		Logger: logger.Named("api.graphql"),
	}
}

// ExtensionName returns the extension name.
func (e *LoggingExtension) ExtensionName() string {
	return "LoggingExtension"
}

// Validate validates the extension configuration.
func (e *LoggingExtension) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

// InterceptResponse logs GraphQL response details.
func (e *LoggingExtension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	start := time.Now()
	oc := graphql.GetOperationContext(ctx)

	resp := next(ctx)

	if resp == nil {
		return resp
	}

	latency := time.Since(start)

	// 基础日志字段
	fields := []zap.Field{
		zap.String("operationName", oc.OperationName),
		zap.Duration("latency", latency),
	}

	// 安全地获取操作类型
	if oc.Operation != nil && oc.Operation.Operation != "" {
		fields = append(fields, zap.String("operation", string(oc.Operation.Operation)))
	}

	// DEBUG 级别：记录请求变量
	if e.Logger.Core().Enabled(zapcore.DebugLevel) {
		fields = append(fields, zap.String("rawQuery", oc.RawQuery))
		fields = append(fields, zap.Any("variables", oc.Variables))
	}

	// 检查是否有错误
	if len(resp.Errors) > 0 {
		// 过滤掉 PersistedQueryNotFound 错误，这不是真正的业务错误
		var filteredErrors []*gqlerror.Error
		for _, err := range resp.Errors {
			if err.Message == "PersistedQueryNotFound" {
				continue
			}
			filteredErrors = append(filteredErrors, err)
		}

		// 只有在过滤后仍有错误时才记录错误日志
		if len(filteredErrors) > 0 {
			errorMsgs := make([]string, len(filteredErrors))
			for i, err := range filteredErrors {
				errorMsgs[i] = err.Message
			}
			fields = append(fields, zap.Strings("errors", errorMsgs))
			e.Logger.Error("GraphQL operation completed with errors", fields...)
		} else {
			// 如果只有 PersistedQueryNotFound 错误，视为正常请求
			if e.Logger.Core().Enabled(zapcore.DebugLevel) {
				fields = append(fields, zap.Any("data", resp.Data))
			}
			e.Logger.Info("GraphQL operation completed", fields...)
		}
	}

	// DEBUG 级别：记录响应数据
	if e.Logger.Core().Enabled(zapcore.DebugLevel) {
		fields = append(fields, zap.Any("data", resp.Data))
	}
	e.Logger.Info("GraphQL operation completed", fields...)

	return resp
}

var _ graphql.HandlerExtension = &LoggingExtension{}
var _ graphql.ResponseInterceptor = &LoggingExtension{}
