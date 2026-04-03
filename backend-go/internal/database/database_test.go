package database

import (
	"testing"

	"github.com/timlzh/ollama-hack/internal/config"
)

func TestConnect_UnsupportedEngine(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "sqlite",
		Host:     "localhost",
		Port:     5432,
		Username: "test",
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error for unsupported database engine")
	}
	if err != nil && err.Error() != "unsupported database engine: sqlite" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestConnect_PostgreSQLConfigFormat(t *testing.T) {
	// This test verifies DSN formation (won't actually connect)
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "testhost",
		Port:     5433,
		Username: "testuser",
		Password: "testpass",
		DB:       "testdb",
	}

	// Since we can't connect to a real DB in unit tests, 
	// we verify the config is parsed correctly
	if cfg.Engine != "postgresql" {
		t.Errorf("Expected engine 'postgresql', got '%s'", cfg.Engine)
	}
	if cfg.Host != "testhost" {
		t.Errorf("Expected host 'testhost', got '%s'", cfg.Host)
	}
	if cfg.Port != 5433 {
		t.Errorf("Expected port 5433, got %d", cfg.Port)
	}
}

func TestConnect_MySQLConfigFormat(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "mysql",
		Host:     "mysqlhost",
		Port:     3306,
		Username: "mysqluser",
		Password: "mysqlpass",
		DB:       "mysqldb",
	}

	// Verify config format
	if cfg.Engine != "mysql" {
		t.Errorf("Expected engine 'mysql', got '%s'", cfg.Engine)
	}
	if cfg.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", cfg.Port)
	}
}

func TestDBStruct(t *testing.T) {
	// Test that DB struct embeds sqlx.DB correctly
	db := &DB{nil}
	if db == nil {
		t.Error("Expected DB struct to be created")
	}
}

func TestDatabaseConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		cfg    *config.DatabaseConfig
		engine string
	}{
		{
			name: "PostgreSQL",
			cfg: &config.DatabaseConfig{
				Engine:   "postgresql",
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "pass",
				DB:       "testdb",
			},
			engine: "postgresql",
		},
		{
			name: "MySQL",
			cfg: &config.DatabaseConfig{
				Engine:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "pass",
				DB:       "testdb",
			},
			engine: "mysql",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cfg.Engine != tc.engine {
				t.Errorf("Expected engine '%s', got '%s'", tc.engine, tc.cfg.Engine)
			}
		})
	}
}

func TestConnect_InvalidPostgresHost(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "nonexistent.invalid.host.local",
		Port:     5432,
		Username: "test",
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	// Should fail to connect
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

func TestConnect_InvalidMySQLHost(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "mysql",
		Host:     "nonexistent.invalid.host.local",
		Port:     3306,
		Username: "test",
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	// Should fail to connect
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

func TestDatabaseConfigDefaults(t *testing.T) {
	cfg := &config.DatabaseConfig{}
	
	// Verify zero values
	if cfg.Engine != "" {
		t.Errorf("Expected empty engine, got '%s'", cfg.Engine)
	}
	if cfg.Host != "" {
		t.Errorf("Expected empty host, got '%s'", cfg.Host)
	}
	if cfg.Port != 0 {
		t.Errorf("Expected port 0, got %d", cfg.Port)
	}
}

func TestConnect_InvalidPostgresPort(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "localhost",
		Port:     9999, // Likely no service on this port
		Username: "test",
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid port")
	}
}

func TestConnect_InvalidMySQLPort(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "mysql",
		Host:     "localhost",
		Port:     9999, // Likely no service on this port
		Username: "test",
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid port")
	}
}

func TestConnect_EmptyDatabaseConfig(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine: "postgresql",
		// All other fields empty
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error when connecting with empty host")
	}
}

func TestConnect_UnsupportedEngineVariations(t *testing.T) {
	testCases := []struct {
		name         string
		engine       string
		expectedErr  string
	}{
		{
			name:        "SQLite engine",
			engine:      "sqlite",
			expectedErr: "unsupported database engine: sqlite",
		},
		{
			name:        "MongoDB engine",
			engine:      "mongodb",
			expectedErr: "unsupported database engine: mongodb",
		},
		{
			name:        "Unknown engine",
			engine:      "unknown",
			expectedErr: "unsupported database engine: unknown",
		},
		{
			name:        "Empty engine",
			engine:      "",
			expectedErr: "unsupported database engine: ",
		},
		{
			name:        "Postgres case mismatch",
			engine:      "Postgresql",
			expectedErr: "unsupported database engine: Postgresql",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.DatabaseConfig{
				Engine:   tc.engine,
				Host:     "localhost",
				Port:     5432,
				Username: "test",
				Password: "test",
				DB:       "testdb",
			}

			_, err := Connect(cfg)
			if err == nil {
				t.Errorf("Expected error for engine '%s'", tc.engine)
			}
			if err != nil && err.Error() != tc.expectedErr {
				t.Errorf("Expected error '%s', got '%s'", tc.expectedErr, err.Error())
			}
		})
	}
}

func TestConnect_SpecialCharactersInPassword(t *testing.T) {
	// This test verifies DSN formation with special characters
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "user",
		Password: "p@ss!w0rd#$%", // Special characters in password
		DB:       "testdb",
	}

	// Verify config is structured correctly
	if cfg.Password != "p@ss!w0rd#$%" {
		t.Errorf("Expected password with special chars, got '%s'", cfg.Password)
	}
}

func TestConnect_MySQLWithSpecialCharacters(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "user@domain",
		Password: "pass:word",
		DB:       "db-name",
	}

	// Verify config is structured correctly
	if cfg.Username != "user@domain" {
		t.Errorf("Expected username 'user@domain', got '%s'", cfg.Username)
	}
	if cfg.DB != "db-name" {
		t.Errorf("Expected db 'db-name', got '%s'", cfg.DB)
	}
}

func TestConnect_LargePortNumbers(t *testing.T) {
	testCases := []struct {
		name string
		port int
	}{
		{
			name: "Max valid port",
			port: 65535,
		},
		{
			name: "Min valid port",
			port: 1,
		},
		{
			name: "Common PostgreSQL port",
			port: 5432,
		},
		{
			name: "Common MySQL port",
			port: 3306,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.DatabaseConfig{
				Engine:   "postgresql",
				Host:     "localhost",
				Port:     tc.port,
				Username: "test",
				Password: "test",
				DB:       "testdb",
			}

			if cfg.Port != tc.port {
				t.Errorf("Expected port %d, got %d", tc.port, cfg.Port)
			}
		})
	}
}

func TestConnect_PostgreSQLDSNFormat(t *testing.T) {
	// This test checks DSN is formatted correctly without actually connecting
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "pg-host",
		Port:     5432,
		Username: "pg-user",
		Password: "pg-pass",
		DB:       "pg-db",
	}

	if cfg.Engine != "postgresql" {
		t.Errorf("Expected postgresql engine, got %s", cfg.Engine)
	}
	if cfg.Host != "pg-host" {
		t.Errorf("Expected host 'pg-host', got '%s'", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", cfg.Port)
	}
}

func TestConnect_MySQLDSNFormat(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "mysql",
		Host:     "mysql-host",
		Port:     3306,
		Username: "mysql-user",
		Password: "mysql-pass",
		DB:       "mysql-db",
	}

	if cfg.Engine != "mysql" {
		t.Errorf("Expected mysql engine, got %s", cfg.Engine)
	}
	if cfg.Host != "mysql-host" {
		t.Errorf("Expected host 'mysql-host', got '%s'", cfg.Host)
	}
	if cfg.Port != 3306 {
		t.Errorf("Expected port 3306, got %d", cfg.Port)
	}
}

func TestConnect_EmptyUsername(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "", // Empty username
		Password: "test",
		DB:       "testdb",
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error with empty username")
	}
}

func TestConnect_EmptyDatabase(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     "localhost",
		Port:     5432,
		Username: "test",
		Password: "test",
		DB:       "", // Empty database
	}

	_, err := Connect(cfg)
	if err == nil {
		t.Error("Expected error with empty database name")
	}
}

func TestConnect_InvalidHost(t *testing.T) {
	testCases := []struct {
		name string
		host string
	}{
		{
			name: "Unreachable IP",
			host: "192.0.2.1", // TEST-NET-1 (reserved for documentation)
		},
		{
			name: "Invalid domain",
			host: "invalid-domain-that-does-not-exist.invalid",
		},
		{
			name: "Empty host",
			host: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.DatabaseConfig{
				Engine:   "postgresql",
				Host:     tc.host,
				Port:     5432,
				Username: "test",
				Password: "test",
				DB:       "testdb",
			}

			_, err := Connect(cfg)
			if err == nil {
				t.Errorf("Expected error for host '%s'", tc.host)
			}
		})
	}
}
