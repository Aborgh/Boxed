package database

import (
	"Boxed/internal/models"
	"errors"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
)

func SetupDatabase() (*gorm.DB, error) {
	var envVariables = [...]string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "DB_TZ"}
	for _, envVariable := range envVariables {
		if os.Getenv(envVariable) == "" && envVariable != "DB_SSLMODE" {
			return nil, errors.New(fmt.Sprintf("%s environment variable not set", envVariable))
		}
		if envVariable == "DB_SSLMODE" {
			err := os.Setenv("DB_SSLMODE", "disable")
			if err != nil {
				return nil, err
			}
		}
	}
	dsn := os.ExpandEnv("host=${DB_HOST} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT} sslmode=${DB_SSLMODE} TimeZone=${DB_TZ}")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(models.Box{}, models.Item{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func CloseDatabase(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Could not get DB instance: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}
}
