package config

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// LogLevels represents hierarchical log level configuration.
// Keys are module paths (e.g., "core.db", "rclone") and values are log levels.
type LogLevels map[string]string

// LogLevelsDecodeHook returns a DecodeHookFunc that skips decoding for LogLevels.
// This is needed because viper.AllSettings() converts dotted keys to nested maps,
// which causes decoding errors. We handle LogLevels separately using viper.Get()
// which preserves the flat map structure.
func LogLevelsDecodeHook() mapstructure.DecodeHookFunc {
	return func(_ reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		// Only handle conversion to LogLevels type
		if to != reflect.TypeOf(LogLevels{}) {
			return data, nil
		}

		// Return an empty LogLevels - it will be populated later via viper.Get()
		return make(LogLevels), nil
	}
}
