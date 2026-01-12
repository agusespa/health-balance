package database

import (
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("Error closing database: %v", err)
		}
	}()

	if db.DB == nil {
		t.Fatal("Database connection is nil")
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM user_profile").Scan(&count)
	if err != nil {
		t.Errorf("Failed to query user_profile table: %v", err)
	}
}

func TestInitWithExistingPath(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "existing.db")

	db1, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	_ = db1.Close()

	db2, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer func() {
		if err := db2.Close(); err != nil {
			t.Errorf("Error closing database: %v", err)
		}
	}()

	if db2.DB == nil {
		t.Fatal("Database connection is nil")
	}
}

func TestInitError(t *testing.T) {
	invalidPath := "/nonexistent/path/test.db"

	_, err := Init(invalidPath)
	if err == nil {
		t.Error("Expected error when initializing with invalid path, but got none")
	}
}
