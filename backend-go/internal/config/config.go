package config

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
}

type AppConfig struct {
	Env                      string `mapstructure:"env"`
	LogLevel                 string `mapstructure:"log_level"`
	SecretKey                string `mapstructure:"secret_key"`
	Algorithm                string `mapstructure:"algorithm"`
	AccessTokenExpireMinutes int    `mapstructure:"access_token_expire_minutes"`
}

type DatabaseConfig struct {
	Engine   string `mapstructure:"engine"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("app.env", "prod")
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.secret_key", "0llama_H4ck")
	viper.SetDefault("app.algorithm", "HS256")
	viper.SetDefault("app.access_token_expire_minutes", 30)
	viper.SetDefault("database.engine", "postgresql")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.username", "ollama_hack")
	viper.SetDefault("database.password", "0llama_H4ck")
	viper.SetDefault("database.db", "ollama_hack")

	// Read config file if exists (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		log.Println("No config file found, using environment variables and defaults")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// BackgroundEndpointOutboundEnabled reports whether background jobs may reach out to
// third-party Ollama endpoints (cyclical testing, discovery scans, Shodan, health probes).
// Default true for backwards compatibility; set BACKGROUND_ENDPOINT_OUTBOUND=false on prod
// when you mainly use cloud and want to avoid broadcasting your egress IP during probes.
func BackgroundEndpointOutboundEnabled() bool {
	val := strings.TrimSpace(os.Getenv("BACKGROUND_ENDPOINT_OUTBOUND"))
	if val == "" {
		return true
	}
	switch strings.ToLower(val) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
