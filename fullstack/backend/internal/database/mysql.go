package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"pharma-platform/internal/config"
)

func NewMySQL(cfg config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	var pingErr error
	for range 20 {
		pingErr = db.Ping()
		if pingErr == nil {
			return db, nil
		}
		time.Sleep(2 * time.Second)
	}

	_ = db.Close()
	return nil, fmt.Errorf("ping mysql after retries: %w", pingErr)
}
