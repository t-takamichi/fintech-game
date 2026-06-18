package db

import (
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	maxOpen := 25
	if mo := os.Getenv("DB_MAX_OPEN_CONNS"); mo != "" {
		if val, err := strconv.Atoi(mo); err == nil {
			maxOpen = val
		}
	}

	maxIdle := 25
	if mi := os.Getenv("DB_MAX_IDLE_CONNS"); mi != "" {
		if val, err := strconv.Atoi(mi); err == nil {
			maxIdle = val
		}
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Minute * 5)

	return gdb, nil
}

func InitDBFromEnv() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("POSTGRES_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("POSTGRES_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("POSTGRES_USER")
		if user == "" {
			user = "bank_user"
		}
		pass := os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			pass = "bank_pass"
		}
		dbname := os.Getenv("POSTGRES_DB")
		if dbname == "" {
			dbname = "bank_db"
		}
		dsn = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	}
	return NewGormDB(dsn)
}
