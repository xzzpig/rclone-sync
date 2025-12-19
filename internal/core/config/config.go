// Package config provides configuration management for the application.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	Rclone struct {
		ConfigPath string `mapstructure:"config_path"`
	} `mapstructure:"rclone"`
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	App struct {
		DataDir     string `mapstructure:"data_dir"`
		Environment string `mapstructure:"environment"`
	} `mapstructure:"app"`
	Security struct {
		EncryptionKey string `mapstructure:"encryption_key"`
	} `mapstructure:"security"`
}

// Cfg is the global configuration instance.
var Cfg Config

// InitConfig initializes the application configuration from file and environment variables.
func InitConfig(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix("CLOUDSYNC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// Only exit if it's not a "config file not found" error
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Error reading config file:", err)
			os.Exit(1)
		}
		// Config file not found is acceptable; continue with defaults
	}

	if err := viper.Unmarshal(&Cfg); err != nil {
		fmt.Println("Unable to decode into struct:", err)
		os.Exit(1)
	}
}

func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("database.path", "cloud-sync.db")
	viper.SetDefault("database.migration_mode", "versioned")
	viper.SetDefault("rclone.config_path", "rclone.conf")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("app.data_dir", "./app_data")
	viper.SetDefault("app.environment", "production")
	viper.SetDefault("security.encryption_key", "")
}

// BindFlags binds command-line flags to configuration values.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("config", "", "config file (default is ./config.toml)")
	cmd.PersistentFlags().Int("port", 8080, "Port to run the server on")
	_ = viper.BindPFlag("server.port", cmd.PersistentFlags().Lookup("port"))
}
