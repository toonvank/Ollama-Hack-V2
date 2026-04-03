//go:build integration
// +build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timlzh/ollama-hack/internal/config"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     host,
		Port:     port.Int(),
		Username: "testuser",
		Password: "testpass",
		DB:       "testdb",
	}

	db, err := Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	cleanup := func() {
		db.Close()
		pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

func TestIntegration_Connect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected db to be non-nil")
	}

	err := db.Ping()
	if err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
}

func TestIntegration_CreateTables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	err := db.CreateTables()
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	// Verify tables exist
	tables := []string{"users", "plans", "api_keys", "endpoints", "ai_models", "settings"}
	for _, table := range tables {
		var exists bool
		err := db.Get(&exists, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)`, table)
		if err != nil {
			t.Fatalf("failed to check if table %s exists: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestIntegration_CreateTablesIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create tables twice - should not error
	err := db.CreateTables()
	if err != nil {
		t.Fatalf("failed to create tables first time: %v", err)
	}

	err = db.CreateTables()
	if err != nil {
		t.Fatalf("failed to create tables second time: %v", err)
	}
}
