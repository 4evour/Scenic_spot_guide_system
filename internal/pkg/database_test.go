package pkg

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/scenic-guide/internal/config"
)

func TestPostgresDSNIncludesConnectionFields(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "postgres",
		Port:     5432,
		Name:     "scenic_guide",
		User:     "scenic",
		Password: "secret",
	}

	dsn := postgresDSN(cfg)

	for _, want := range []string{
		"host=postgres",
		"port=5432",
		"user=scenic",
		"password=secret",
		"dbname=scenic_guide",
		"sslmode=disable",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("postgres dsn missing %q: %s", want, dsn)
		}
	}
}

func TestConnectionPoolDefaults(t *testing.T) {
	cfg := &config.DatabaseConfig{}

	maxOpen, maxIdle, lifetime := connectionPoolSettings(cfg)

	if maxOpen != defaultMaxOpenConns {
		t.Fatalf("max open = %d, want %d", maxOpen, defaultMaxOpenConns)
	}
	if maxIdle != defaultMaxIdleConns {
		t.Fatalf("max idle = %d, want %d", maxIdle, defaultMaxIdleConns)
	}
	if lifetime != time.Duration(defaultConnMaxLifetimeMinutes)*time.Minute {
		t.Fatalf("lifetime = %s, want %d minutes", lifetime, defaultConnMaxLifetimeMinutes)
	}
}

func TestInitDatabaseRejectsUnsupportedDriver(t *testing.T) {
	err := InitDatabase(&config.DatabaseConfig{Driver: "oracle"})
	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
	if !strings.Contains(err.Error(), "unsupported database driver") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitDatabaseCreatesSQLiteParentDirectory(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "scenic.db")

	if err := InitDatabase(&config.DatabaseConfig{Driver: "sqlite", Path: dbPath}); err != nil {
		t.Fatalf("init sqlite database: %v", err)
	}

	if DB == nil {
		t.Fatal("expected global database handle to be initialized")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	if _, err := sqlDB.Exec("CREATE TABLE IF NOT EXISTS path_check (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("write sqlite database: %v", err)
	}
}
