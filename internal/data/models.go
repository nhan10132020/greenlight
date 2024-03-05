package data

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Movies MovieModel
}

func NewModels(db *gorm.DB) Models {
	return Models{
		Movies: MovieModel{
			DB: db,
		},
	}
}
