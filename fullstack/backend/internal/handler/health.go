package handler

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) HealthHandler {
	return HealthHandler{db: db}
}

func (h HealthHandler) GetHealth(c *gin.Context) {
	dbStatus := "up"
	status := "ok"

	if err := h.db.Ping(); err != nil {
		dbStatus = "down"
		status = "degraded"
	}

	writeSuccess(c, http.StatusOK, gin.H{
		"status":   status,
		"database": dbStatus,
		"service":  "go-backend",
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}
