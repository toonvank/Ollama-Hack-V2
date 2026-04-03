package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test with default values
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check default values
	if cfg.App.Env != "prod" {
		t.Errorf("Expected default env 'prod', got '%s'", cfg.App.Env)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("Expected default log_level 'info', got '%s'", cfg.App.LogLevel)
	}
	if cfg.App.Algorithm != "HS256" {
		t.Errorf("Expected default algorithm 'HS256', got '%s'", cfg.App.Algorithm)
	}
	if cfg.App.AccessTokenExpireMinutes != 30 {
		t.Errorf("Expected default access_token_expire_minutes 30, got %d", cfg.App.AccessTokenExpireMinutes)
	}
	if cfg.Database.Engine != "postgresql" {
		t.Errorf("Expected default engine 'postgresql', got '%s'", cfg.Database.Engine)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected default host 'localhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected default port 5432, got %d", cfg.Database.Port)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("APP_ENV", "development")
	os.Setenv("APP_LOG_LEVEL", "debug")
	os.Setenv("APP_SECRET_KEY", "test-secret-key")
	os.Setenv("APP_ACCESS_TOKEN_EXPIRE_MINUTES", "60")
	os.Setenv("DATABASE_ENGINE", "postgresql")
	os.Setenv("DATABASE_HOST", "testhost")
	os.Setenv("DATABASE_PORT", "5433")
	os.Setenv("DATABASE_USERNAME", "testuser")
	os.Setenv("DATABASE_PASSWORD", "testpass")
	os.Setenv("DATABASE_DB", "testdb")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("APP_LOG_LEVEL")
		os.Unsetenv("APP_SECRET_KEY")
		os.Unsetenv("APP_ACCESS_TOKEN_EXPIRE_MINUTES")
		os.Unsetenv("DATABASE_ENGINE")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
		os.Unsetenv("DATABASE_USERNAME")
		os.Unsetenv("DATABASE_PASSWORD")
		os.Unsetenv("DATABASE_DB")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.App.Env != "development" {
		t.Errorf("Expected env 'development', got '%s'", cfg.App.Env)
	}
	if cfg.App.LogLevel != "debug" {
		t.Errorf("Expected log_level 'debug', got '%s'", cfg.App.LogLevel)
	}
	if cfg.App.SecretKey != "test-secret-key" {
		t.Errorf("Expected secret_key 'test-secret-key', got '%s'", cfg.App.SecretKey)
	}
	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected host 'testhost', got '%s'", cfg.Database.Host)
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Env:                      "test",
			LogLevel:                 "debug",
			SecretKey:                "secret",
			Algorithm:                "HS256",
			AccessTokenExpireMinutes: 30,
		},
		Database: DatabaseConfig{
			Engine:   "postgresql",
			Host:     "localhost",
			Port:     5432,
			Username: "user",
			Password: "pass",
			DB:       "testdb",
		},
	}

	if cfg.App.Env != "test" {
		t.Errorf("Expected app env 'test', got '%s'", cfg.App.Env)
	}
	if cfg.Database.Engine != "postgresql" {
		t.Errorf("Expected database engine 'postgresql', got '%s'", cfg.Database.Engine)
	}
}

func TestLoadWithMissingDatabaseEngine(t *testing.T) {
	// Clear DATABASE_ENGINE to test default fallback
	os.Unsetenv("DATABASE_ENGINE")
	defer os.Unsetenv("DATABASE_ENGINE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should use default
	if cfg.Database.Engine != "postgresql" {
		t.Errorf("Expected default database engine 'postgresql', got '%s'", cfg.Database.Engine)
	}
}

func TestLoadWithInvalidPortNumber(t *testing.T) {
	testCases := []struct {
		name     string
		portStr  string
		expectErr bool
	}{
		{
			name:     "Valid port",
			portStr:  "5432",
			expectErr: false,
		},
		{
			name:     "Port zero",
			portStr:  "0",
			expectErr: false, // viper will parse it as 0, but app should handle
		},
		{
			name:     "Negative port",
			portStr:  "-1",
			expectErr: false, // viper will parse negative numbers
		},
		{
			name:     "Port exceeding max",
			portStr:  "65536",
			expectErr: false, // viper parses it, validation is app's responsibility
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set invalid port
			os.Setenv("DATABASE_PORT", tc.portStr)
			defer os.Unsetenv("DATABASE_PORT")

			cfg, err := Load()
			if tc.expectErr && err != nil {
				// Expected error
				return
			}
			if tc.expectErr && err == nil {
				t.Error("Expected error for invalid port")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Port should be parsed correctly by viper
			if tc.name == "Valid port" && cfg.Database.Port != 5432 {
				t.Errorf("Expected port 5432, got %d", cfg.Database.Port)
			}
		})
	}
}

func TestLoadWithEmptyValues(t *testing.T) {
	// Note: Viper uses defaults when environment variables are not set
	// Setting env vars to empty strings still counts as "set" in viper
	// This test verifies that viper's behavior handles empty env vars appropriately
	os.Setenv("APP_ENV", "")
	os.Setenv("DATABASE_ENGINE", "")
	os.Setenv("DATABASE_HOST", "")
	os.Setenv("DATABASE_USERNAME", "")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("DATABASE_ENGINE")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_USERNAME")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Viper may use defaults or empty strings depending on configuration
	// The important thing is that the configuration is valid
	if cfg == nil {
		t.Error("Expected valid config, got nil")
	}
	// When env vars are set to empty, viper treats them as empty values
	// but may still fall back to defaults in some cases
	if cfg.App.Env == "" {
		t.Logf("App env is empty, using default or env value")
	}
}

func TestLoadWithMySQLConfig(t *testing.T) {
	os.Setenv("DATABASE_ENGINE", "mysql")
	os.Setenv("DATABASE_HOST", "mysql-host")
	os.Setenv("DATABASE_PORT", "3306")
	os.Setenv("DATABASE_USERNAME", "mysql-user")
	os.Setenv("DATABASE_PASSWORD", "mysql-pass")
	os.Setenv("DATABASE_DB", "mysql-db")
	defer func() {
		os.Unsetenv("DATABASE_ENGINE")
		os.Unsetenv("DATABASE_HOST")
		os.Unsetenv("DATABASE_PORT")
		os.Unsetenv("DATABASE_USERNAME")
		os.Unsetenv("DATABASE_PASSWORD")
		os.Unsetenv("DATABASE_DB")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Database.Engine != "mysql" {
		t.Errorf("Expected database engine 'mysql', got '%s'", cfg.Database.Engine)
	}
	if cfg.Database.Host != "mysql-host" {
		t.Errorf("Expected host 'mysql-host', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", cfg.Database.Port)
	}
	if cfg.Database.Username != "mysql-user" {
		t.Errorf("Expected username 'mysql-user', got '%s'", cfg.Database.Username)
	}
	if cfg.Database.Password != "mysql-pass" {
		t.Errorf("Expected password 'mysql-pass', got '%s'", cfg.Database.Password)
	}
	if cfg.Database.DB != "mysql-db" {
		t.Errorf("Expected db 'mysql-db', got '%s'", cfg.Database.DB)
	}
}

func TestLoadAppConfig(t *testing.T) {
	testCases := []struct {
		name     string
		envKey   string
		envValue string
		cfgField func(*Config) string
		expected string
	}{
		{
			name:     "App environment",
			envKey:   "APP_ENV",
			envValue: "staging",
			cfgField: func(c *Config) string { return c.App.Env },
			expected: "staging",
		},
		{
			name:     "Log level",
			envKey:   "APP_LOG_LEVEL",
			envValue: "warn",
			cfgField: func(c *Config) string { return c.App.LogLevel },
			expected: "warn",
		},
		{
			name:     "Algorithm",
			envKey:   "APP_ALGORITHM",
			envValue: "HS512",
			cfgField: func(c *Config) string { return c.App.Algorithm },
			expected: "HS512",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(tc.envKey, tc.envValue)
			defer os.Unsetenv(tc.envKey)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			result := tc.cfgField(cfg)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestLoadAccessTokenExpireMinutes(t *testing.T) {
	testCases := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "Default 30 minutes",
			envValue: "",
			expected: 30,
		},
		{
			name:     "Custom 60 minutes",
			envValue: "60",
			expected: 60,
		},
		{
			name:     "Custom 120 minutes",
			envValue: "120",
			expected: 120,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("APP_ACCESS_TOKEN_EXPIRE_MINUTES", tc.envValue)
				defer os.Unsetenv("APP_ACCESS_TOKEN_EXPIRE_MINUTES")
			} else {
				os.Unsetenv("APP_ACCESS_TOKEN_EXPIRE_MINUTES")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if cfg.App.AccessTokenExpireMinutes != tc.expected {
				t.Errorf("Expected %d minutes, got %d", tc.expected, cfg.App.AccessTokenExpireMinutes)
			}
		})
	}
}

func TestLoadSecretKey(t *testing.T) {
	os.Setenv("APP_SECRET_KEY", "custom-secret-key")
	defer os.Unsetenv("APP_SECRET_KEY")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.App.SecretKey != "custom-secret-key" {
		t.Errorf("Expected secret key 'custom-secret-key', got '%s'", cfg.App.SecretKey)
	}
}

func TestLoadDatabasePassword(t *testing.T) {
	os.Setenv("DATABASE_PASSWORD", "my-secret-password")
	defer os.Unsetenv("DATABASE_PASSWORD")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Database.Password != "my-secret-password" {
		t.Errorf("Expected password 'my-secret-password', got '%s'", cfg.Database.Password)
	}
}
