package pkg

import (
	"database/sql"
	"os"

	"github.com/scenic-guide/internal/config"
	sqlite3 "modernc.org/sqlite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	sql.Register("modernc", &sqlite3.Driver{})
}

func InitDatabase(cfg *config.DatabaseConfig) error {
	var err error

	if cfg.Driver == "sqlite" {
		os.MkdirAll("./data", os.ModePerm)

		DB, err = gorm.Open(sqlite.New(sqlite.Config{
			DriverName: "modernc",
			DSN:        cfg.Path,
		}), &gorm.Config{})
	}

	if err != nil {
		return err
	}

	return nil
}
