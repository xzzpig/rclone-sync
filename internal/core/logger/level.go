// Package logger provides logging utilities for the application.
package logger

import (
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

// levelCache 使用 sync.Map 实现无锁并发缓存
// Key: logger name (string), Value: zapcore.Level
var levelCache sync.Map

// levelConfig 存储层级日志级别配置
var (
	levelConfigMu  sync.RWMutex
	levelConfigMap map[string]string // 配置的层级级别映射
	globalLevel    zapcore.Level     // 全局默认级别
)

// InitLevelConfig 初始化层级日志级别配置
// 在 InitLogger 中调用，传入配置文件中的 levels map
func InitLevelConfig(levels map[string]string, defaultLevel zapcore.Level) {
	levelConfigMu.Lock()
	defer levelConfigMu.Unlock()
	levelConfigMap = levels
	globalLevel = defaultLevel
	// 清空缓存，因为配置已变更
	levelCache = sync.Map{}
}

// GetLevelForName 根据日志名称查找最匹配的日志级别
// 使用按需缓存策略：首次计算后缓存，后续直接查表
// 匹配过程区分大小写
func GetLevelForName(name string) zapcore.Level {
	// 1. 先查缓存
	if cached, ok := levelCache.Load(name); ok {
		return cached.(zapcore.Level)
	}

	// 2. 计算匹配的级别
	level := computeLevelForName(name)

	// 3. 存入缓存
	levelCache.Store(name, level)

	return level
}

// computeLevelForName 计算日志名称对应的级别（不使用缓存）
func computeLevelForName(name string) zapcore.Level {
	levelConfigMu.RLock()
	defer levelConfigMu.RUnlock()

	if len(levelConfigMap) == 0 {
		return globalLevel
	}

	// 空字符串直接返回全局级别
	if name == "" {
		return globalLevel
	}

	// 1. 精确匹配
	if levelStr, ok := levelConfigMap[name]; ok {
		if level, err := ParseLevel(levelStr); err == nil {
			return level
		}
		// 无效级别值，继续尝试父级匹配
	}

	// 2. 按 "." 拆分后逐级向上匹配父级
	parts := strings.Split(name, ".")
	for i := len(parts) - 1; i > 0; i-- {
		prefix := strings.Join(parts[:i], ".")
		if levelStr, ok := levelConfigMap[prefix]; ok {
			if level, err := ParseLevel(levelStr); err == nil {
				return level
			}
			// 无效级别值，继续尝试更高层级
		}
	}

	// 3. 返回全局级别
	return globalLevel
}

// ParseLevel 解析日志级别字符串（不区分大小写）
// 支持: debug, info, warn, error
func ParseLevel(levelStr string) (zapcore.Level, error) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(strings.ToLower(levelStr)))
	return level, err
}
