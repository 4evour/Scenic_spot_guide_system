package handler

import (
	"errors"

	"gorm.io/gorm"
)

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
