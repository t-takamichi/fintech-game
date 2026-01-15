package db

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewGormDB opens a GORM DB connection and returns the DB handle.
func NewGormDB(dsn string) (*gorm.DB, error) {
	dialector := postgres.Open(dsn)
	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(time.Minute * 5)

	// NOTE: schema migrations are managed by SQL migration files and the migrate tool.
	// Do NOT call AutoMigrate here to avoid schema drift in production.

	return gdb, nil
}
