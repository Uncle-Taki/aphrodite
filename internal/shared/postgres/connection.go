package postgres

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
)

func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	logLevel := gormlogger.Silent
	if config.C.Debug {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	logger.L.Info("connected to postgres")
	return db, nil
}
