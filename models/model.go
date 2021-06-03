package models

import (
	"gitlab.com/zsmartex/finex/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Lock() (tx *gorm.DB) {
	return config.DataBase.Clauses(clause.Locking{Strength: "UPDATE"})
}

type Reference struct {
	ID   uint64
	Type string
}
