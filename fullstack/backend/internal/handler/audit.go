package handler

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (a *API) ListAuditLogs(c *gin.Context) {
	moduleFilter := strings.TrimSpace(c.Query("module"))
	actionFilter := strings.TrimSpace(c.Query("action"))
	userIDFilter := strings.TrimSpace(c.Query("user_id"))
	queryFilter := strings.TrimSpace(c.Query("q"))
	fromFilter := strings.TrimSpace(c.Query("from"))
	toFilter := strings.TrimSpace(c.Query("to"))

	whereParts := []string{"1=1"}
	args := make([]any, 0)

	if moduleFilter != "" {
		whereParts = append(whereParts, "al.module_name = ?")
		args = append(args, moduleFilter)
	}
	if actionFilter != "" {
		whereParts = append(whereParts, "al.action = ?")
		args = append(args, actionFilter)
	}
	if userIDFilter != "" {
		uid, err := strconv.ParseInt(userIDFilter, 10, 64)
		if err != nil {
			badRequest(c, "INVALID_USER_ID", "user_id must be numeric")
			return
		}
		whereParts = append(whereParts, "al.user_id = ?")
		args = append(args, uid)
	}
	if queryFilter != "" {
		like := "%" + strings.ToLower(queryFilter) + "%"
		whereParts = append(whereParts, "(LOWER(al.record_id) LIKE ? OR LOWER(CAST(al.details_json AS CHAR)) LIKE ?)")
		args = append(args, like, like)
	}
	if fromFilter != "" {
		fromTime, err := time.Parse("2006-01-02", fromFilter)
		if err != nil {
			badRequest(c, "INVALID_FROM", "from must be YYYY-MM-DD")
			return
		}
		whereParts = append(whereParts, "al.created_at >= ?")
		args = append(args, fromTime)
	}
	if toFilter != "" {
		toTime, err := time.Parse("2006-01-02", toFilter)
		if err != nil {
			badRequest(c, "INVALID_TO", "to must be YYYY-MM-DD")
			return
		}
		whereParts = append(whereParts, "al.created_at < ?")
		args = append(args, toTime.Add(24*time.Hour))
	}

	page := parsePositiveInt(c.Query("page"), 1)
	size := parsePositiveInt(c.Query("size"), 20)
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size

	whereClause := strings.Join(whereParts, " AND ")
	rows, err := a.db.Query(`
		SELECT al.id, al.user_id, COALESCE(u.username, ''), al.action, al.module_name, al.record_id,
		       COALESCE(CAST(al.details_json AS CHAR), ''), COALESCE(al.ip_address, ''), COALESCE(al.user_agent, ''), al.created_at
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		WHERE `+whereClause+`
		ORDER BY al.id DESC
		LIMIT ? OFFSET ?
	`, append(args, size, offset)...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list audit logs")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id        int64
			userID    int64
			username  string
			action    string
			module    string
			recordID  string
			details   string
			ip        string
			userAgent string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &userID, &username, &action, &module, &recordID, &details, &ip, &userAgent, &createdAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan audit logs")
			return
		}
		items = append(items, gin.H{
			"id":          id,
			"user_id":     userID,
			"username":    username,
			"action":      action,
			"module_name": module,
			"record_id":   recordID,
			"details":     details,
			"ip_address":  ip,
			"user_agent":  userAgent,
			"created_at":  createdAt.Format(time.RFC3339),
		})
	}

	var total int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM audit_logs al WHERE "+whereClause, args...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to count audit logs")
		return
	}

	writeSuccess(c, http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"size":  size,
		"total": total,
	})
}

func (a *API) ExportAuditLogs(c *gin.Context) {
	moduleFilter := strings.TrimSpace(c.Query("module"))
	actionFilter := strings.TrimSpace(c.Query("action"))

	whereParts := []string{"1=1"}
	args := make([]any, 0)
	if moduleFilter != "" {
		whereParts = append(whereParts, "module_name = ?")
		args = append(args, moduleFilter)
	}
	if actionFilter != "" {
		whereParts = append(whereParts, "action = ?")
		args = append(args, actionFilter)
	}

	rows, err := a.db.Query(`
		SELECT id, user_id, action, module_name, record_id, COALESCE(CAST(details_json AS CHAR), ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at
		FROM audit_logs
		WHERE `+strings.Join(whereParts, " AND ")+`
		ORDER BY id DESC
		LIMIT 20000
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to export audit logs")
		return
	}
	defer rows.Close()

	buf := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buf)
	_ = writer.Write([]string{"id", "user_id", "action", "module_name", "record_id", "details", "ip_address", "user_agent", "created_at"})

	for rows.Next() {
		var (
			id        int64
			userID    int64
			action    string
			module    string
			recordID  string
			details   string
			ip        string
			userAgent string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &userID, &action, &module, &recordID, &details, &ip, &userAgent, &createdAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan audit rows")
			return
		}
		_ = writer.Write([]string{
			strconv.FormatInt(id, 10),
			strconv.FormatInt(userID, 10),
			action,
			module,
			recordID,
			details,
			ip,
			userAgent,
			createdAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeError(c, http.StatusInternalServerError, "CSV_ERROR", "failed to generate csv")
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	c.String(http.StatusOK, buf.String())
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
