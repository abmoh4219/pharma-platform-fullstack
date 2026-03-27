package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pharma-platform/internal/middleware"
)

type ComplianceService struct {
	db *sql.DB
}

func NewComplianceService(db *sql.DB) *ComplianceService {
	return &ComplianceService{db: db}
}

type RestrictionRule struct {
	ID                    int64
	MedName               string
	RuleType              string
	MaxQuantity           float64
	RequiresApproval      bool
	RequiresPrescription  bool
	MinIntervalDays       int
	FeeAmount             float64
	FeeCurrency           string
	IsActive              bool
	Institution           string
	Department            string
	Team                  string
}

type PurchaseCheckInput struct {
	MedName                   string
	Quantity                  float64
	ClientID                  string
	PrescriptionAttachmentID  int64
	RecordPurchase            bool
}

type PurchaseCheckResult struct {
	RuleID                 int64   `json:"rule_id"`
	RuleType               string  `json:"rule_type"`
	MaxQuantity            float64 `json:"max_quantity"`
	RequiresApproval       bool    `json:"requires_approval"`
	RequiresPrescription   bool    `json:"requires_prescription"`
	MinIntervalDays        int     `json:"min_interval_days"`
	FeeAmount              float64 `json:"fee_amount"`
	FeeCurrency            string  `json:"fee_currency"`
	Allowed                bool    `json:"allowed"`
	Reason                 string  `json:"reason"`
	Recorded               bool    `json:"recorded"`
	PurchaseRecordID       int64   `json:"purchase_record_id"`
}

type PurchaseRecord struct {
	RestrictionID            int64
	ClientID                 string
	MedName                  string
	Quantity                 float64
	PrescriptionAttachmentID int64
	Details                  map[string]any
}

func (s *ComplianceService) LoadRestrictionByMed(ctx context.Context, user middleware.AuthUser, medName string) (RestrictionRule, error) {
	med := strings.TrimSpace(medName)
	if med == "" {
		return RestrictionRule{}, sql.ErrNoRows
	}

	where, args := middleware.BuildScopeWhere(user, "")
	query := `
		SELECT id, med_name, rule_type, max_quantity, requires_approval, requires_prescription,
		       min_interval_days, fee_amount, fee_currency, is_active, institution, department, team
		FROM restrictions
		WHERE med_name = ? AND ` + where + `
		ORDER BY id DESC
		LIMIT 1`
	queryArgs := append([]any{med}, args...)

	var out RestrictionRule
	err := s.db.QueryRowContext(ctx, query, queryArgs...).Scan(
		&out.ID,
		&out.MedName,
		&out.RuleType,
		&out.MaxQuantity,
		&out.RequiresApproval,
		&out.RequiresPrescription,
		&out.MinIntervalDays,
		&out.FeeAmount,
		&out.FeeCurrency,
		&out.IsActive,
		&out.Institution,
		&out.Department,
		&out.Team,
	)
	if err != nil {
		return RestrictionRule{}, err
	}

	if out.MinIntervalDays <= 0 {
		out.MinIntervalDays = 7
	}
	if strings.TrimSpace(out.FeeCurrency) == "" {
		out.FeeCurrency = "USD"
	}

	return out, nil
}

func (s *ComplianceService) CheckPurchaseRestriction(ctx context.Context, user middleware.AuthUser, in PurchaseCheckInput) (PurchaseCheckResult, error) {
	med := strings.TrimSpace(in.MedName)
	clientID := strings.TrimSpace(in.ClientID)
	if med == "" || in.Quantity <= 0 {
		return PurchaseCheckResult{}, fmt.Errorf("med_name and positive quantity are required")
	}
	if clientID == "" {
		return PurchaseCheckResult{}, fmt.Errorf("client_id is required")
	}

	rule, err := s.LoadRestrictionByMed(ctx, user, med)
	if err != nil {
		if err == sql.ErrNoRows {
			return PurchaseCheckResult{Allowed: true, Reason: "no active rule"}, nil
		}
		return PurchaseCheckResult{}, err
	}

	result := PurchaseCheckResult{
		RuleID:               rule.ID,
		RuleType:             rule.RuleType,
		MaxQuantity:          rule.MaxQuantity,
		RequiresApproval:     rule.RequiresApproval,
		RequiresPrescription: rule.RequiresPrescription,
		MinIntervalDays:      rule.MinIntervalDays,
		FeeAmount:            rule.FeeAmount,
		FeeCurrency:          rule.FeeCurrency,
		Allowed:              true,
		Reason:               "within rule",
	}

	if !rule.IsActive {
		result.Reason = "rule is inactive"
		return result, nil
	}

	if in.Quantity > rule.MaxQuantity {
		result.Allowed = false
		result.Reason = fmt.Sprintf("quantity %.2f exceeds max %.2f", in.Quantity, rule.MaxQuantity)
		return result, nil
	}

	if rule.RequiresPrescription {
		if in.PrescriptionAttachmentID <= 0 {
			result.Allowed = false
			result.Reason = "prescription attachment is required"
			return result, nil
		}
		ok, err := s.validatePrescriptionAttachment(ctx, user, in.PrescriptionAttachmentID)
		if err != nil {
			return PurchaseCheckResult{}, err
		}
		if !ok {
			result.Allowed = false
			result.Reason = "invalid prescription attachment"
			return result, nil
		}
	}

	blocked, err := s.hasRecentClientPurchase(ctx, user, clientID, rule.ID, rule.MinIntervalDays)
	if err != nil {
		return PurchaseCheckResult{}, err
	}
	if blocked {
		result.Allowed = false
		result.Reason = fmt.Sprintf("client purchase blocked: once per %d days", rule.MinIntervalDays)
		return result, nil
	}

	if rule.RequiresApproval {
		result.Reason = "requires approval"
	}

	if in.RecordPurchase && result.Allowed {
		recordID, err := s.RecordPurchase(ctx, user, PurchaseRecord{
			RestrictionID:            rule.ID,
			ClientID:                 clientID,
			MedName:                  med,
			Quantity:                 in.Quantity,
			PrescriptionAttachmentID: in.PrescriptionAttachmentID,
			Details: map[string]any{
				"requires_approval": rule.RequiresApproval,
				"fee_amount":        rule.FeeAmount,
				"fee_currency":      rule.FeeCurrency,
			},
		})
		if err != nil {
			return PurchaseCheckResult{}, err
		}
		result.Recorded = true
		result.PurchaseRecordID = recordID
	}

	return result, nil
}

func (s *ComplianceService) validatePrescriptionAttachment(ctx context.Context, user middleware.AuthUser, attachmentID int64) (bool, error) {
	var (
		moduleName  string
		recordID    int64
	)
	if err := s.db.QueryRowContext(ctx, `
		SELECT module_name, record_id
		FROM attachments
		WHERE id = ?
	`, attachmentID).Scan(&moduleName, &recordID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if recordID <= 0 {
		return false, nil
	}

	moduleName = strings.TrimSpace(moduleName)
	if moduleName != "qualifications" && moduleName != "case_ledgers" {
		return false, nil
	}
	if user.Role == "system_admin" {
		return true, nil
	}

	if moduleName == "qualifications" {
		where, args := middleware.BuildScopeWhere(user, "q")
		queryArgs := append([]any{recordID}, args...)
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM qualifications q WHERE q.id = ? AND `+where, queryArgs...).Scan(&count); err != nil {
			return false, err
		}
		return count > 0, nil
	}

	where, args := middleware.BuildScopeWhere(user, "cl")
	queryArgs := append([]any{recordID}, args...)
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM case_ledgers cl WHERE cl.id = ? AND `+where, queryArgs...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *ComplianceService) hasRecentClientPurchase(ctx context.Context, user middleware.AuthUser, clientID string, restrictionID int64, minIntervalDays int) (bool, error) {
	if minIntervalDays <= 0 {
		minIntervalDays = 7
	}

	where, args := middleware.BuildScopeWhere(user, "cpr")
	queryArgs := []any{clientID, restrictionID, minIntervalDays}
	queryArgs = append(queryArgs, args...)

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM compliance_purchase_records cpr
		WHERE cpr.client_id = ?
		  AND cpr.restriction_id = ?
		  AND cpr.created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL ? DAY)
		  AND `+where+`
	`, queryArgs...).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *ComplianceService) RecordPurchase(ctx context.Context, user middleware.AuthUser, record PurchaseRecord) (int64, error) {
	detailsJSON := "{}"
	if record.Details != nil {
		buf, err := json.Marshal(record.Details)
		if err == nil {
			detailsJSON = string(buf)
		}
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO compliance_purchase_records
		(restriction_id, client_id, med_name, quantity, prescription_attachment_id, institution, department, team, reviewed_by, details_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.RestrictionID, strings.TrimSpace(record.ClientID), strings.TrimSpace(record.MedName), record.Quantity,
		record.PrescriptionAttachmentID, user.Institution, user.Department, user.Team, user.ID, detailsJSON)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *ComplianceService) RestrictionBeforeAfter(ctx context.Context, id int64) (map[string]any, error) {
	var (
		medName              string
		ruleType             string
		maxQuantity          float64
		requiresApproval     bool
		requiresPrescription bool
		minIntervalDays      int
		feeAmount            float64
		feeCurrency          string
		isActive             bool
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT med_name, rule_type, max_quantity, requires_approval, requires_prescription, min_interval_days, fee_amount, fee_currency, is_active
		FROM restrictions
		WHERE id = ?
	`, id).Scan(&medName, &ruleType, &maxQuantity, &requiresApproval, &requiresPrescription, &minIntervalDays, &feeAmount, &feeCurrency, &isActive)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"med_name":              medName,
		"rule_type":             ruleType,
		"max_quantity":          maxQuantity,
		"requires_approval":     requiresApproval,
		"requires_prescription": requiresPrescription,
		"min_interval_days":     minIntervalDays,
		"fee_amount":            feeAmount,
		"fee_currency":          feeCurrency,
		"is_active":             isActive,
	}, nil
}

func (s *ComplianceService) NormalizeNow(value *time.Time) time.Time {
	if value == nil {
		return time.Now().UTC()
	}
	return value.UTC()
}
