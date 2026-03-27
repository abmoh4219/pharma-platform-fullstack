package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"pharma-platform/internal/middleware"
)

type CaseService struct {
	db *sql.DB
}

func NewCaseService(db *sql.DB) *CaseService {
	return &CaseService{db: db}
}

type CaseHistoryRecord struct {
	ID         int64
	CaseID      int64
	ActionType  string
	FromStatus  *string
	ToStatus    *string
	Note        *string
	AssignedTo  *int64
	Details     map[string]any
	Institution string
	Department  string
	Team        string
	ChangedBy   int64
	CreatedAt   time.Time
}

func (s *CaseService) RecordHistory(
	ctx context.Context,
	user middleware.AuthUser,
	caseID int64,
	actionType string,
	fromStatus *string,
	toStatus *string,
	note *string,
	assignedTo *int64,
	details map[string]any,
) (int64, error) {
	detailsJSON := "{}"
	if details != nil {
		buf, _ := json.Marshal(details)
		detailsJSON = string(buf)
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO case_processing_records
		(case_id, action_type, from_status, to_status, note, assigned_to, details_json, institution, department, team, changed_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, caseID, actionType, nullableString(fromStatus), nullableString(toStatus), nullableString(note), nullableInt64(assignedTo), detailsJSON,
		user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *CaseService) ListHistory(ctx context.Context, user middleware.AuthUser, caseID int64) ([]CaseHistoryRecord, error) {
	if caseID <= 0 {
		return nil, fmt.Errorf("invalid case id")
	}
	where, scopeArgs := middleware.BuildScopeWhere(user, "cl")
	checkArgs := append([]any{caseID}, scopeArgs...)
	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM case_ledgers cl
		WHERE cl.id = ? AND `+where+`
	`, checkArgs...).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, sql.ErrNoRows
	}

	whereHist, histArgs := middleware.BuildScopeWhere(user, "cpr")
	args := append([]any{caseID}, histArgs...)
	rows, err := s.db.QueryContext(ctx, `
		SELECT cpr.id, cpr.case_id, cpr.action_type, cpr.from_status, cpr.to_status, cpr.note,
		       cpr.assigned_to, COALESCE(CAST(cpr.details_json AS CHAR), '{}'), cpr.institution, cpr.department, cpr.team,
		       cpr.changed_by, cpr.created_at
		FROM case_processing_records cpr
		WHERE cpr.case_id = ? AND `+whereHist+`
		ORDER BY cpr.id DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CaseHistoryRecord, 0)
	for rows.Next() {
		var (
			item        CaseHistoryRecord
			fromStatus  sql.NullString
			toStatus    sql.NullString
			note        sql.NullString
			assignedTo  sql.NullInt64
			detailsRaw  string
		)
		if err := rows.Scan(&item.ID, &item.CaseID, &item.ActionType, &fromStatus, &toStatus, &note, &assignedTo, &detailsRaw,
			&item.Institution, &item.Department, &item.Team, &item.ChangedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		if fromStatus.Valid {
			v := fromStatus.String
			item.FromStatus = &v
		}
		if toStatus.Valid {
			v := toStatus.String
			item.ToStatus = &v
		}
		if note.Valid {
			v := note.String
			item.Note = &v
		}
		if assignedTo.Valid {
			v := assignedTo.Int64
			item.AssignedTo = &v
		}
		_ = json.Unmarshal([]byte(detailsRaw), &item.Details)
		out = append(out, item)
	}
	return out, nil
}

func nullableString(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}

func nullableInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
