package config

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"PoolManagerVM/backend/models"
)

var DB *gorm.DB

func Sync_DB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("PoolManagerVM.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	DB.AutoMigrate(&models.User{})
}
