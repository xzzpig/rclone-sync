// Package logger provides logging utilities for the application.
package logger

import (
	"go.uber.org/zap/zapcore"
)

// levelFilterCore 是一个包装的 zapcore.Core，用于过滤日志级别
type levelFilterCore struct {
	zapcore.Core
	level zapcore.Level
}

// Enabled 检查给定级别是否应该被记录
func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
	return lvl >= c.level
}

// Check 实现 zapcore.Core 接口，使用自定义的级别过滤
// 必须覆盖此方法，因为嵌入类型的 Check() 方法会调用嵌入类型自己的 Enabled()，
// 而不是外层 levelFilterCore 覆盖的 Enabled() 方法
func (c *levelFilterCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

var (
	_ zapcore.Core         = (*levelFilterCore)(nil)
	_ zapcore.LevelEnabler = (*levelFilterCore)(nil)
)
