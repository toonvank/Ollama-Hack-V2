package database

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/timlzh/ollama-hack/internal/config"
)

type DB struct {
	*sqlx.DB
}

func Connect(cfg *config.DatabaseConfig) (*DB, error) {
	var dsn string

	switch cfg.Engine {
	case "postgresql":
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DB,
		)
	case "mysql":
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.DB,
		)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", cfg.Engine)
	}

	engine := cfg.Engine
	if engine == "postgresql" {
		engine = "postgres"
	}
	db, err := sqlx.Connect(engine, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)

	log.Printf("Connected to %s database at %s:%d", engine, cfg.Host, cfg.Port)

	return &DB{db}, nil
}

func (db *DB) CreateTables() error {
	schema := `
	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		hashed_password VARCHAR(255) NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		plan_id INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Plans table
	CREATE TABLE IF NOT EXISTS plans (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		description TEXT,
		rpm INTEGER NOT NULL,
		rpd INTEGER NOT NULL,
		is_default BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- API Keys table
	CREATE TABLE IF NOT EXISTS api_keys (
		id SERIAL PRIMARY KEY,
		key VARCHAR(128) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		last_used_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Endpoints table
	CREATE TABLE IF NOT EXISTS endpoints (
		id SERIAL PRIMARY KEY,
		url VARCHAR(512) UNIQUE NOT NULL,
		name VARCHAR(255) NOT NULL,
		status VARCHAR(50) DEFAULT 'pending',
		endpoint_type VARCHAR(50) DEFAULT 'ollama',
		api_key TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Endpoint test tasks table
	CREATE TABLE IF NOT EXISTS endpoint_test_tasks (
		id SERIAL PRIMARY KEY,
		endpoint_id INTEGER NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
		status VARCHAR(50) DEFAULT 'pending',
		scheduled_at TIMESTAMP NOT NULL,
		last_tried TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Models table
	CREATE TABLE IF NOT EXISTS ai_models (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		tag VARCHAR(255) NOT NULL,
		enabled BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, tag)
	);

	-- Endpoint AI Models junction table
	CREATE TABLE IF NOT EXISTS endpoint_ai_models (
		id SERIAL PRIMARY KEY,
		endpoint_id INTEGER NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
		ai_model_id INTEGER NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,
		status VARCHAR(50) DEFAULT 'available',
		token_per_second FLOAT,
		max_connection_time FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(endpoint_id, ai_model_id)
	);

	-- Endpoint Performances table
	CREATE TABLE IF NOT EXISTS endpoint_performances (
		id SERIAL PRIMARY KEY,
		endpoint_id INTEGER NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
		status VARCHAR(50) NOT NULL,
		ollama_version VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Model Performance table
	CREATE TABLE IF NOT EXISTS ai_model_performances (
		id SERIAL PRIMARY KEY,
		endpoint_ai_model_id INTEGER NOT NULL REFERENCES endpoint_ai_models(id) ON DELETE CASCADE,
		token_per_second FLOAT,
		max_connection_time FLOAT,
		total_time FLOAT DEFAULT 120,
		output TEXT,
		output_tokens INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Settings table
	CREATE TABLE IF NOT EXISTS settings (
		id SERIAL PRIMARY KEY,
		key VARCHAR(255) UNIQUE NOT NULL,
		value TEXT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- API Key Usage table
	CREATE TABLE IF NOT EXISTS api_key_usage (
		id SERIAL PRIMARY KEY,
		api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
		endpoint VARCHAR(512) NOT NULL,
		method VARCHAR(10) NOT NULL,
		status_code INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Endpoint Health table
	CREATE TABLE IF NOT EXISTS endpoint_health (
		url VARCHAR(512) PRIMARY KEY,
		score INTEGER NOT NULL,
		success_count INTEGER DEFAULT 0,
		fail_count INTEGER DEFAULT 0,
		disabled BOOLEAN DEFAULT FALSE,
		disabled_until TIMESTAMP,
		last_success TIMESTAMP,
		last_fail TIMESTAMP,
		last_probe TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
	CREATE INDEX IF NOT EXISTS idx_endpoints_status ON endpoints(status);
	CREATE INDEX IF NOT EXISTS idx_endpoint_test_tasks_endpoint_id ON endpoint_test_tasks(endpoint_id);
	CREATE INDEX IF NOT EXISTS idx_endpoint_test_tasks_status ON endpoint_test_tasks(status);
	CREATE INDEX IF NOT EXISTS idx_endpoint_ai_models_endpoint_id ON endpoint_ai_models(endpoint_id);
	CREATE INDEX IF NOT EXISTS idx_endpoint_ai_models_ai_model_id ON endpoint_ai_models(ai_model_id);
	CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
	CREATE INDEX IF NOT EXISTS idx_api_key_usage_created_at ON api_key_usage(created_at);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Migrations for schema changes
	migrations := `
	-- Expand api_keys.key column from VARCHAR(64) to VARCHAR(128) to accommodate longer keys
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'api_keys' 
			AND column_name = 'key' 
			AND character_maximum_length = 64
		) THEN
			ALTER TABLE api_keys ALTER COLUMN key TYPE VARCHAR(128);
		END IF;
	END $$;
	`

	_, err = db.Exec(migrations)
	if err != nil {
		log.Printf("Warning: Some migrations failed: %v", err)
		// Don't return error, as tables are already created
	}

	log.Println("Database tables created successfully")
	return nil
}
