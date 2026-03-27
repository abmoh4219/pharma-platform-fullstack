package app

import (
	"fmt"

	"pharma-platform/internal/config"
	"pharma-platform/internal/database"
	"pharma-platform/internal/logging"
	"pharma-platform/internal/router"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.NewMySQL(cfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	if err := database.ApplyInitSQL(db, "migrations/init.sql"); err != nil {
		return fmt.Errorf("apply init schema: %w", err)
	}

	r := router.New(cfg, db)
	addr := fmt.Sprintf(":%d", cfg.AppPort)

	logging.Info("app", "backend listening", map[string]any{"addr": addr})
	return r.Run(addr)
}
