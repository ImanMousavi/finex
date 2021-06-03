package config

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DataBase *gorm.DB

func ConnectDatabase() error {
	dsn := "host=" + os.Getenv("DATABASE_HOST") +
		" user=" + os.Getenv("DATABASE_USER") +
		" password=" + os.Getenv("DATABASE_PASS") +
		" dbname=" + os.Getenv("DATABASE_NAME") +
		" port=" + os.Getenv("DATABASE_PORT") +
		" sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})

	if err != nil {
		return err
	}

	DataBase = db

	return nil
}
