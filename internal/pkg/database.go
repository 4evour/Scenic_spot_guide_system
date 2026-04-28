package pkg

import (
	"os"

	"github.com/scenic-guide/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase(cfg *config.DatabaseConfig) error {
	var err error

	if cfg.Driver == "sqlite" {
		os.MkdirAll("./data", os.ModePerm)
		DB, err = gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	}

	if err != nil {
		return err
	}

	return nil
}
