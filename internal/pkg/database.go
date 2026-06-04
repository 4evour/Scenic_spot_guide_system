package pkg

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scenic-guide/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	sqlite3 "modernc.org/sqlite"
)

// db 是包级别的数据库连接实例，通过 InitDatabase 初始化
var db *gorm.DB

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return db
}

const (
	defaultMaxOpenConns           = 25
	defaultMaxIdleConns           = 10
	defaultConnMaxLifetimeMinutes = 30
	defaultPostgresPort           = 5432
	defaultPostgresSSLMode        = "require"
	defaultSQLitePath             = "./data/scenic_guide.db"
)

func init() {
	sql.Register("modernc", &sqlite3.Driver{})
}

func InitDatabase(cfg *config.DatabaseConfig) error {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if driver == "" {
		driver = "postgres"
	}

	var err error
	switch driver {
	case "postgres", "postgresql":
		db, err = gorm.Open(postgres.Open(postgresDSN(cfg)), &gorm.Config{})
	case "sqlite":
		path := strings.TrimSpace(cfg.Path)
		if path == "" {
			path = defaultSQLitePath
		}
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return err
			}
		}

		db, err = gorm.Open(sqlite.New(sqlite.Config{
			DriverName: "modernc",
			DSN:        path,
		}), &gorm.Config{})
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	maxOpen, maxIdle, maxLifetime := connectionPoolSettings(cfg)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(maxLifetime)

	return nil
}

func postgresDSN(cfg *config.DatabaseConfig) string {
	host := valueOrDefault(cfg.Host, "127.0.0.1")
	port := cfg.Port
	if port == 0 {
		port = defaultPostgresPort
	}
	name := valueOrDefault(cfg.Name, "scenic_guide")
	user := valueOrDefault(cfg.User, "scenic")

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		host,
		port,
		user,
		cfg.Password,
		name,
		defaultPostgresSSLMode,
	)
}

func connectionPoolSettings(cfg *config.DatabaseConfig) (int, int, time.Duration) {
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = defaultMaxOpenConns
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = defaultMaxIdleConns
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	lifetimeMinutes := cfg.ConnMaxLifetimeMinutes
	if lifetimeMinutes <= 0 {
		lifetimeMinutes = defaultConnMaxLifetimeMinutes
	}
	return maxOpen, maxIdle, time.Duration(lifetimeMinutes) * time.Minute
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
