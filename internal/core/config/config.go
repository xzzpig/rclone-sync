// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xzzpig/rclone-sync/internal/core/errs"
)

// Config represents the application configuration structure.
type Config struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`
	Database struct {
		Path          string `mapstructure:"path"`
		MigrationMode string `mapstructure:"migration_mode"`
	} `mapstructure:"database"`
	Log struct {
		Level  string    `mapstructure:"level"`
		Levels LogLevels `mapstructure:"levels"`
	} `mapstructure:"log"`
	App struct {
		DataDir     string `mapstructure:"data_dir"`
		Environment string `mapstructure:"environment"`
		Job         struct {
			AutoDeleteEmptyJobs  bool   `mapstructure:"auto_delete_empty_jobs"`
			MaxLogsPerConnection int    `mapstructure:"max_logs_per_connection"`
			CleanupSchedule      string `mapstructure:"cleanup_schedule"`
		} `mapstructure:"job"`
		Sync struct {
			Transfers int `mapstructure:"transfers"` // Default parallel transfers (1-64), default: 4
		} `mapstructure:"sync"`
	} `mapstructure:"app"`
	Security struct {
		EncryptionKey string `mapstructure:"encryption_key"`
	} `mapstructure:"security"`
	Auth struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	} `mapstructure:"auth"`
}

// Load loads the application configuration from file and environment variables.
// Returns the configuration and any error encountered.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix("RCLONESYNC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// Only return error if it's not a "config file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found is acceptable; continue with defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			LogLevelsDecodeHook(),
		),
	)); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	// Workaround for viper issue: viper.AllSettings() (used by Unmarshal) converts
	// dotted keys to nested maps, which causes conflicts when both parent and child
	// keys exist (e.g., "api" and "api.graphql"). The child key gets lost.
	// We need to directly get the log.levels value which preserves the flat map structure.
	if rawLevels := viper.Get("log.levels"); rawLevels != nil {
		if levelsMap, ok := rawLevels.(map[string]interface{}); ok {
			cfg.Log.Levels = make(LogLevels)
			for k, v := range levelsMap {
				if strVal, ok := v.(string); ok {
					cfg.Log.Levels[k] = strVal
				} else {
					cfg.Log.Levels[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return &cfg, nil
}

// IsAuthEnabled returns true if authentication is configured
func (c *Config) IsAuthEnabled() bool {
	return c.Auth.Username != "" && c.Auth.Password != ""
}

// ValidateAuth checks if auth configuration is valid
func (c *Config) ValidateAuth() error {
	hasUsername := c.Auth.Username != ""
	hasPassword := c.Auth.Password != ""
	if hasUsername != hasPassword {
		return errs.ConstError("username and password must both be set or both be empty")
	}
	return nil
}

func setDefaults() {
	// 通过反射自动注册所有配置键，使 AutomaticEnv() 能正确识别环境变量
	registerConfigKeys(reflect.TypeOf(Config{}), "")

	// 设置非零默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.path", "rclone-sync.db")
	viper.SetDefault("database.migration_mode", "versioned")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("app.data_dir", "./app_data")
	viper.SetDefault("app.environment", "production")
	viper.SetDefault("app.job.auto_delete_empty_jobs", true)
	viper.SetDefault("app.job.max_logs_per_connection", 1000)
	viper.SetDefault("app.job.cleanup_schedule", "0 * * * *")
	viper.SetDefault("app.sync.transfers", 4)
}

// registerConfigKeys 通过反射遍历结构体，为每个字段注册零值默认值
// 这样 viper.AutomaticEnv() 就能识别所有配置键对应的环境变量
func registerConfigKeys(t reflect.Type, prefix string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}

		kind := field.Type.Kind()

		switch kind {
		case reflect.Struct:
			// 如果是结构体，递归处理
			registerConfigKeys(field.Type, key)
		case reflect.Map:
			// 跳过 map 类型，因为它们需要特殊处理（如 LogLevels）
			continue
		default:
			// 注册零值默认值，使 viper 知道这个键存在
			viper.SetDefault(key, reflect.Zero(field.Type).Interface())
		}
	}
}

// BindFlags binds command-line flags to configuration values.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Int("port", 8080, "Port to run the server on")
	_ = viper.BindPFlag("server.port", cmd.PersistentFlags().Lookup("port"))
}
