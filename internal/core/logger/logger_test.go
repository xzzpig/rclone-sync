// Package logger provides logging utilities for the application.
package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNamedLogger_UsesCorrectLevel(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 创建基础 logger
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	logger = testLogger
	defer func() { logger = originalLogger }()

	// 设置层级日志级别配置
	InitLevelConfig(map[string]string{
		"core.db":        "debug",
		"core.scheduler": "warn",
	}, zapcore.InfoLevel)

	// 测试 Named logger 使用正确级别
	// core.db 应该使用 debug 级别
	dbLogger := Named("core.db")
	assert.NotNil(t, dbLogger)

	// core.scheduler 应该使用 warn 级别
	schedulerLogger := Named("core.scheduler")
	assert.NotNil(t, schedulerLogger)

	// api 应该使用全局 info 级别
	apiLogger := Named("api")
	assert.NotNil(t, apiLogger)
}

func TestNamedLogger_LevelFiltering(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 创建基础 logger（允许所有级别）
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	logger = testLogger
	defer func() { logger = originalLogger }()

	// 设置层级日志级别配置
	// core.db 使用 warn 级别（只允许 warn 和 error）
	InitLevelConfig(map[string]string{
		"core.db": "warn",
	}, zapcore.InfoLevel)

	// 获取 named logger
	dbLogger := Named("core.db")
	require.NotNil(t, dbLogger)

	// 清空 buffer
	buf.Reset()

	// Debug 和 Info 级别日志应该被过滤掉
	dbLogger.Debug("debug message - should be filtered")
	dbLogger.Info("info message - should be filtered")

	// Warn 和 Error 级别日志应该被记录
	dbLogger.Warn("warn message - should be logged")
	dbLogger.Error("error message - should be logged")

	// 检查输出
	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestNamedLogger_GlobalLevelFiltering(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 创建基础 logger（允许所有级别）
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	logger = testLogger
	defer func() { logger = originalLogger }()

	// 设置全局级别为 error，无特定配置
	InitLevelConfig(map[string]string{}, zapcore.ErrorLevel)

	// 获取 named logger
	apiLogger := Named("api.graphql")
	require.NotNil(t, apiLogger)

	// 清空 buffer
	buf.Reset()

	// Debug, Info, Warn 级别日志应该被过滤掉
	apiLogger.Debug("debug message - should be filtered")
	apiLogger.Info("info message - should be filtered")
	apiLogger.Warn("warn message - should be filtered")

	// 只有 Error 级别日志应该被记录
	apiLogger.Error("error message - should be logged")

	// 检查输出
	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestNamedLogger_ParentLevelInheritance(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 创建基础 logger（允许所有级别）
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	logger = testLogger
	defer func() { logger = originalLogger }()

	// 设置 core 为 debug 级别
	InitLevelConfig(map[string]string{
		"core": "debug",
	}, zapcore.ErrorLevel)

	// 获取子级 named logger - 应该继承 core 的 debug 级别
	dbLogger := Named("core.db")
	require.NotNil(t, dbLogger)

	// 清空 buffer
	buf.Reset()

	// 所有级别的日志都应该被记录（因为继承了 debug 级别）
	dbLogger.Debug("debug message - should be logged")
	dbLogger.Info("info message - should be logged")
	dbLogger.Warn("warn message - should be logged")
	dbLogger.Error("error message - should be logged")

	// 检查输出
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestInitLogger_DevelopmentEnvironment(t *testing.T) {
	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// 测试 development 环境初始化
	InitLogger(EnvironmentDevelopment, LogLevelDebug, map[string]string{
		"core.db": "warn",
	})

	// 验证 logger 已初始化
	assert.NotNil(t, logger)

	// 验证 Named logger 可以正常工作
	dbLogger := Named("core.db")
	assert.NotNil(t, dbLogger)
}

func TestInitLogger_ProductionEnvironment(t *testing.T) {
	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// 测试 production 环境初始化
	InitLogger(EnvironmentProduction, Info, map[string]string{})

	// 验证 logger 已初始化
	assert.NotNil(t, logger)
}

func TestInitLogger_WithLevelsConfig(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// 初始化 logger，设置全局级别为 error，但 core.db 为 debug
	InitLogger(EnvironmentProduction, Error, map[string]string{
		"core.db": "debug",
	})

	// 替换 logger 的 core 为我们的测试 buffer
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	logger = zap.New(core)

	// 重新初始化层级配置（因为我们替换了 logger）
	InitLevelConfig(map[string]string{
		"core.db": "debug",
	}, zapcore.ErrorLevel)

	// 获取 named logger
	dbLogger := Named("core.db")
	apiLogger := Named("api")

	// 清空 buffer
	buf.Reset()

	// core.db 应该使用 debug 级别，允许所有日志
	dbLogger.Debug("db debug - should be logged")

	// api 应该使用全局 error 级别
	apiLogger.Debug("api debug - should be filtered")
	apiLogger.Error("api error - should be logged")

	// 检查输出
	output := buf.String()
	assert.Contains(t, output, "db debug")
	assert.NotContains(t, output, "api debug")
	assert.Contains(t, output, "api error")
}

func TestInitLogger_LogLevelMapping(t *testing.T) {
	tests := []struct {
		logLevel    LogLevel
		expectedZap zapcore.Level
	}{
		{LogLevelDebug, zapcore.DebugLevel},
		{Info, zapcore.InfoLevel},
		{Warn, zapcore.WarnLevel},
		{Error, zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(string(tt.logLevel), func(t *testing.T) {
			result := getZapLevel(string(tt.logLevel))
			assert.Equal(t, tt.expectedZap, result)
		})
	}
}

func TestInitLogger_DefaultLogLevel(t *testing.T) {
	// 测试无效或未知的日志级别默认为 info
	result := getZapLevel("unknown")
	assert.Equal(t, zapcore.InfoLevel, result)

	result = getZapLevel("")
	assert.Equal(t, zapcore.InfoLevel, result)
}

func TestInitLogger_NilLevelsMap(t *testing.T) {
	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// 测试 nil levels map 不会导致 panic
	assert.NotPanics(t, func() {
		InitLogger(EnvironmentProduction, Info, nil)
	})

	// 验证 logger 已初始化
	assert.NotNil(t, logger)
}

func TestNamedLogger_DifferentModulesIndependent(t *testing.T) {
	// 创建一个测试用的 buffer 来捕获日志输出
	var buf bytes.Buffer
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 创建基础 logger（允许所有级别）
	core := zapcore.NewCore(encoder, zapcore.AddSync(&buf), zapcore.DebugLevel)
	testLogger := zap.New(core)

	// 保存原始 logger 并在测试后恢复
	originalLogger := logger
	logger = testLogger
	defer func() { logger = originalLogger }()

	// 设置不同模块的级别
	InitLevelConfig(map[string]string{
		"core.db":        "debug",
		"core.scheduler": "error",
	}, zapcore.InfoLevel)

	// 获取两个不同的 named logger
	dbLogger := Named("core.db")
	schedulerLogger := Named("core.scheduler")

	// 清空 buffer
	buf.Reset()

	// core.db 允许 debug，core.scheduler 只允许 error
	dbLogger.Debug("db debug - should be logged")
	schedulerLogger.Debug("scheduler debug - should be filtered")
	schedulerLogger.Error("scheduler error - should be logged")

	// 检查输出
	output := buf.String()
	assert.Contains(t, output, "db debug")
	assert.NotContains(t, output, "scheduler debug")
	assert.Contains(t, output, "scheduler error")
}
