package database_test

import (
	"path/filepath"
	"testing"

	"rojgarbridge-backend/database"
)

func TestInitDBAndMigrations(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_init.db")
	dbConn, err := database.InitDB(tempDB, "")
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer dbConn.Close()

	if dbConn.DriverName() != "sqlite" {
		t.Fatalf("Expected sqlite driver, got %s", dbConn.DriverName())
	}

	var count int
	err = dbConn.QueryRow("SELECT COUNT(1) FROM jobs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query jobs table: %v", err)
	}

	err = dbConn.QueryRow("SELECT COUNT(1) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query users table: %v", err)
	}
}
