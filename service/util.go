package service

import (
	"fmt"
	"newshub-rss-service/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

// GetDb return db connection
func GetDb(cfg model.Config) *gorm.DB {
	if db != nil {
		return db
	}

	if cfg.Driver == "sqlite3" {
		sqliteDB, err := gorm.Open(sqlite.Open(cfg.ConnectionString), &gorm.Config{})
		if err != nil {
			panic("open db error: " + err.Error())
		}

		db = sqliteDB
		return db
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.DbHost, cfg.DbUser, cfg.DbPassword, cfg.DbName, cfg.DbPort,
	)

	pgdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("open db error: " + err.Error())
	}

	// pgdb.DB().SetMaxIdleConns(10)
	// pgdb.DB().SetMaxOpenConns(100)
	// pgdb.DB().SetConnMaxLifetime(time.Hour)

	db = pgdb

	return db
}
