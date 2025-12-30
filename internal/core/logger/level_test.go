// Package logger provides logging utilities for the application.
package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		levelStr  string
		want      zapcore.Level
		wantError bool
	}{
		// 正常情况 - 小写
		{name: "debug lowercase", levelStr: "debug", want: zapcore.DebugLevel, wantError: false},
		{name: "info lowercase", levelStr: "info", want: zapcore.InfoLevel, wantError: false},
		{name: "warn lowercase", levelStr: "warn", want: zapcore.WarnLevel, wantError: false},
		{name: "error lowercase", levelStr: "error", want: zapcore.ErrorLevel, wantError: false},

		// 正常情况 - 大写
		{name: "DEBUG uppercase", levelStr: "DEBUG", want: zapcore.DebugLevel, wantError: false},
		{name: "INFO uppercase", levelStr: "INFO", want: zapcore.InfoLevel, wantError: false},
		{name: "WARN uppercase", levelStr: "WARN", want: zapcore.WarnLevel, wantError: false},
		{name: "ERROR uppercase", levelStr: "ERROR", want: zapcore.ErrorLevel, wantError: false},

		// 正常情况 - 混合大小写
		{name: "Debug mixed", levelStr: "Debug", want: zapcore.DebugLevel, wantError: false},
		{name: "Info mixed", levelStr: "Info", want: zapcore.InfoLevel, wantError: false},
		{name: "Warn mixed", levelStr: "Warn", want: zapcore.WarnLevel, wantError: false},
		{name: "Error mixed", levelStr: "Error", want: zapcore.ErrorLevel, wantError: false},

		// 无效级别
		{name: "invalid level", levelStr: "invalid", want: zapcore.InfoLevel, wantError: true},
		{name: "empty string returns info", levelStr: "", want: zapcore.InfoLevel, wantError: false}, // zap treats empty as info
		{name: "warning is invalid", levelStr: "warning", want: zapcore.WarnLevel, wantError: false}, // zap supports "warning" as alias for "warn"
		{name: "trace unsupported", levelStr: "trace", want: zapcore.InfoLevel, wantError: true},
		{name: "fatal unsupported", levelStr: "fatal", want: zapcore.FatalLevel, wantError: false}, // zap 支持 fatal
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.levelStr)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetLevelForName_ExactMatch(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"core.db.query": "debug",
		"core.db":       "info",
		"core":          "warn",
	}, zapcore.ErrorLevel)

	// 精确匹配测试
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db.query"))
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("core.db"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("core"))
}

func TestGetLevelForName_ParentMatch(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"core.db": "debug",
		"core":    "info",
	}, zapcore.ErrorLevel)

	// 父级匹配测试 - core.db.query 应该匹配 core.db
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db.query"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db.connection"))

	// 更高父级匹配测试 - core.scheduler.task 应该匹配 core
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("core.scheduler"))
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("core.scheduler.task"))
}

func TestGetLevelForName_GlobalFallback(t *testing.T) {
	// 设置测试配置 - 无匹配项
	InitLevelConfig(map[string]string{
		"rclone": "debug",
	}, zapcore.WarnLevel)

	// 全局回退测试 - api 应该使用全局级别
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("api"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("api.graphql"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("unknown.module"))
}

func TestGetLevelForName_CaseSensitive(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"Core.DB": "debug",
		"core.db": "info",
	}, zapcore.ErrorLevel)

	// 大小写敏感测试 - 名称必须完全匹配
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("Core.DB"))
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("core.db"))
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("CORE.DB")) // 无匹配，使用全局
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("Core.db")) // 无匹配，使用全局
}

func TestGetLevelForName_EmptyName(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"core": "debug",
	}, zapcore.InfoLevel)

	// 空字符串名称测试 - 应该返回全局级别
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName(""))
}

func TestGetLevelForName_InvalidLevelValue(t *testing.T) {
	// 设置测试配置 - 包含无效级别值
	InitLevelConfig(map[string]string{
		"core.db": "invalid_level",
		"core":    "debug",
	}, zapcore.InfoLevel)

	// 无效级别值测试 - 应该跳过无效配置，继续匹配父级
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db.query"))
}

func TestGetLevelForName_EmptyConfig(t *testing.T) {
	// 设置空配置
	InitLevelConfig(nil, zapcore.WarnLevel)

	// 空配置测试 - 应该返回全局级别
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("any.name"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("core.db"))

	// 空 map 配置
	InitLevelConfig(map[string]string{}, zapcore.ErrorLevel)
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("any.name"))
}

func TestGetLevelForName_CacheBehavior(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"core.db": "debug",
	}, zapcore.InfoLevel)

	// 第一次调用 - 计算并缓存
	level1 := GetLevelForName("core.db.query")
	assert.Equal(t, zapcore.DebugLevel, level1)

	// 第二次调用 - 应该使用缓存
	level2 := GetLevelForName("core.db.query")
	assert.Equal(t, zapcore.DebugLevel, level2)

	// 更改配置后，缓存应该被清空
	InitLevelConfig(map[string]string{
		"core.db": "warn",
	}, zapcore.InfoLevel)

	// 应该使用新配置
	level3 := GetLevelForName("core.db.query")
	assert.Equal(t, zapcore.WarnLevel, level3)
}

func TestGetLevelForName_Concurrency(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"core.db":        "debug",
		"core.scheduler": "warn",
		"rclone":         "error",
	}, zapcore.InfoLevel)

	// 并发测试 - 确保线程安全
	var wg sync.WaitGroup
	numGoroutines := 100
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			names := []string{
				"core.db.query",
				"core.scheduler.task",
				"rclone.sync",
				"api.graphql",
			}
			for _, name := range names {
				_ = GetLevelForName(name)
			}
		}(i)
	}

	wg.Wait()

	// 验证结果正确
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("core.db.query"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("core.scheduler.task"))
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("rclone.sync"))
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("api.graphql"))
}

func TestGetLevelForName_DeepHierarchy(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"a":       "error",
		"a.b":     "warn",
		"a.b.c":   "info",
		"a.b.c.d": "debug",
	}, zapcore.ErrorLevel)

	// 深层级匹配测试
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("a"))
	assert.Equal(t, zapcore.WarnLevel, GetLevelForName("a.b"))
	assert.Equal(t, zapcore.InfoLevel, GetLevelForName("a.b.c"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("a.b.c.d"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("a.b.c.d.e"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("a.b.c.d.e.f"))
}

func TestGetLevelForName_SingleComponent(t *testing.T) {
	// 设置测试配置
	InitLevelConfig(map[string]string{
		"rclone": "debug",
	}, zapcore.ErrorLevel)

	// 单组件名称测试
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("rclone"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("rclone.sync"))
	assert.Equal(t, zapcore.DebugLevel, GetLevelForName("rclone.about"))
	assert.Equal(t, zapcore.ErrorLevel, GetLevelForName("other"))
}
