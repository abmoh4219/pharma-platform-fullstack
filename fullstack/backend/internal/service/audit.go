package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

type AuditService struct {
	db *sql.DB
}

func NewAuditService(db *sql.DB) *AuditService {
	return &AuditService{db: db}
}

type AuditEvent struct {
	UserID      int64
	Category    string
	Level       string
	Action      string
	ModuleName  string
	RecordID    string
	Before      any
	After       any
	Details     any
	IPAddress   string
	UserAgent   string
}

func (s *AuditService) Log(ctx context.Context, event AuditEvent) error {
	if strings.TrimSpace(event.Category) == "" {
		event.Category = "general"
	}
	if strings.TrimSpace(event.Level) == "" {
		event.Level = "INFO"
	}
	beforeJSON := mustJSON(event.Before)
	afterJSON := mustJSON(event.After)
	detailsJSON := mustJSON(event.Details)
	diffJSON := mustJSON(diffMap(event.Before, event.After))
	ip := normalizeIP(event.IPAddress)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs
		(user_id, category, level, action, module_name, record_id, before_json, after_json, diff_json, details_json, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.UserID, strings.TrimSpace(event.Category), strings.ToUpper(strings.TrimSpace(event.Level)), strings.TrimSpace(event.Action),
		strings.TrimSpace(event.ModuleName), strings.TrimSpace(event.RecordID), beforeJSON, afterJSON, diffJSON, detailsJSON, ip, clip(event.UserAgent, 255))
	return err
}

func normalizeIP(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(v)
	if err == nil {
		return host
	}
	return v
}

func mustJSON(value any) string {
	if value == nil {
		return "null"
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("{\"marshal_error\":%q}", err.Error())
	}
	return string(buf)
}

func toStringMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	buf, err := json.Marshal(value)
	if err != nil {
		return map[string]any{"raw": fmt.Sprint(value)}
	}
	out := map[string]any{}
	if err := json.Unmarshal(buf, &out); err != nil {
		return map[string]any{"raw": string(buf)}
	}
	return out
}

func diffMap(before any, after any) map[string]any {
	from := toStringMap(before)
	to := toStringMap(after)
	keys := map[string]struct{}{}
	for k := range from {
		keys[k] = struct{}{}
	}
	for k := range to {
		keys[k] = struct{}{}
	}
	changes := map[string]any{}
	for k := range keys {
		bVal, bOK := from[k]
		aVal, aOK := to[k]
		if !bOK && aOK {
			changes[k] = map[string]any{"before": nil, "after": aVal}
			continue
		}
		if bOK && !aOK {
			changes[k] = map[string]any{"before": bVal, "after": nil}
			continue
		}
		if fmt.Sprint(bVal) != fmt.Sprint(aVal) {
			changes[k] = map[string]any{"before": bVal, "after": aVal}
		}
	}
	return changes
}

func clip(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= max {
		return trimmed
	}
	return trimmed[:max]
}
