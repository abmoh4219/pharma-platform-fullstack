package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
	"pharma-platform/internal/service"
)

type qualificationRequest struct {
	EntityType        string `json:"entity_type" binding:"required,oneof=client supplier"`
	EntityName        string `json:"entity_name" binding:"required,min=1,max=128"`
	QualificationCode string `json:"qualification_code" binding:"required,min=1,max=128"`
	IssueDate         string `json:"issue_date" binding:"required,datetime=2006-01-02"`
	ExpiryDate        string `json:"expiry_date" binding:"required,datetime=2006-01-02"`
	Status            string `json:"status" binding:"omitempty,oneof=active inactive"`
	Notes             string `json:"notes" binding:"max=2000"`
}

func (a *API) CreateQualification(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req qualificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid qualification payload")
		return
	}

	issueDate, err := parseDate(req.IssueDate)
	if err != nil {
		badRequest(c, "INVALID_ISSUE_DATE", "issue_date must be YYYY-MM-DD")
		return
	}
	expiryDate, err := parseDate(req.ExpiryDate)
	if err != nil {
		badRequest(c, "INVALID_EXPIRY_DATE", "expiry_date must be YYYY-MM-DD")
		return
	}
	if expiryDate.Before(issueDate) {
		badRequest(c, "INVALID_DATE_RANGE", "expiry_date must be greater than or equal to issue_date")
		return
	}

	notesEnc, err := a.cipher.Encrypt(strings.TrimSpace(req.Notes))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt notes")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	res, err := a.db.Exec(`
		INSERT INTO qualifications
		(entity_type, entity_name, qualification_code, issue_date, expiry_date, status, notes_enc, institution, department, team, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.EntityType, strings.TrimSpace(req.EntityName), strings.TrimSpace(req.QualificationCode), issueDate.Format("2006-01-02"), expiryDate.Format("2006-01-02"), status,
		notesEnc, user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create qualification")
		return
	}
	id, _ := res.LastInsertId()
	after, _ := a.loadQualificationForAudit(c.Request.Context(), id)
	a.logAuditDetailed(c, user.ID, "compliance", "INFO", "compliance.qualification.create", "qualifications", strconv.FormatInt(id, 10), nil, after, req)
	writeSuccess(c, http.StatusCreated, gin.H{"id": id})
}

func (a *API) autoDeactivateQualifications() {
	_, _ = a.db.Exec(`
		UPDATE qualifications
		SET status = 'inactive'
		WHERE status != 'inactive' AND expiry_date < CURDATE()
	`)
}

func (a *API) ListQualifications(c *gin.Context) {
	a.autoDeactivateQualifications()
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")

	rows, err := a.db.Query(`
		SELECT id, entity_type, entity_name, qualification_code, issue_date, expiry_date, status, notes_enc,
		       institution, department, team, created_at, updated_at
		FROM qualifications
		WHERE `+where+`
		ORDER BY expiry_date ASC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list qualifications")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	now := time.Now().UTC()
	for rows.Next() {
		var (
			id                int64
			entityType        string
			entityName        string
			qualificationCode string
			issueDate         time.Time
			expiryDate        time.Time
			status            string
			notesEnc          sql.NullString
			institution       string
			department        string
			team              string
			createdAt         time.Time
			updatedAt         time.Time
		)
		if err := rows.Scan(&id, &entityType, &entityName, &qualificationCode, &issueDate, &expiryDate, &status, &notesEnc, &institution, &department, &team, &createdAt, &updatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan qualifications")
			return
		}

		notesMasked := ""
		if notesEnc.Valid && notesEnc.String != "" {
			notesRaw, err := a.cipher.Decrypt(notesEnc.String)
			if err == nil {
				notesMasked = security.MaskText(notesRaw)
			}
		}
		daysToExpiry := int(expiryDate.Sub(now).Hours() / 24)
		highlightRed := daysToExpiry <= 30

		items = append(items, gin.H{
			"id":                 id,
			"entity_type":        entityType,
			"entity_name":        entityName,
			"qualification_code": qualificationCode,
			"issue_date":         issueDate.Format("2006-01-02"),
			"expiry_date":        expiryDate.Format("2006-01-02"),
			"status":             status,
			"notes":              notesMasked,
			"days_to_expiry":     daysToExpiry,
			"highlight_red":      highlightRed,
			"institution":        institution,
			"department":         department,
			"team":               team,
			"created_at":         createdAt.Format(time.RFC3339),
			"updated_at":         updatedAt.Format(time.RFC3339),
		})
	}

	writeSuccess(c, http.StatusOK, items)
}

func (a *API) UpdateQualification(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid qualification id")
		return
	}

	var req qualificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid qualification payload")
		return
	}

	issueDate, err := parseDate(req.IssueDate)
	if err != nil {
		badRequest(c, "INVALID_ISSUE_DATE", "issue_date must be YYYY-MM-DD")
		return
	}
	expiryDate, err := parseDate(req.ExpiryDate)
	if err != nil {
		badRequest(c, "INVALID_EXPIRY_DATE", "expiry_date must be YYYY-MM-DD")
		return
	}
	if expiryDate.Before(issueDate) {
		badRequest(c, "INVALID_DATE_RANGE", "expiry_date must be greater than or equal to issue_date")
		return
	}

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	checkArgs := []any{id}
	checkArgs = append(checkArgs, scopeArgs...)
	var exists int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM qualifications WHERE id = ? AND "+where, checkArgs...).Scan(&exists); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate qualification")
		return
	}
	if exists == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "qualification not found in your scope")
		return
	}

	before, _ := a.loadQualificationForAudit(c.Request.Context(), id)

	notesEnc, err := a.cipher.Encrypt(strings.TrimSpace(req.Notes))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt notes")
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	_, err = a.db.Exec(`
		UPDATE qualifications
		SET entity_type = ?, entity_name = ?, qualification_code = ?, issue_date = ?, expiry_date = ?, status = ?, notes_enc = ?
		WHERE id = ?
	`, req.EntityType, strings.TrimSpace(req.EntityName), strings.TrimSpace(req.QualificationCode), issueDate.Format("2006-01-02"), expiryDate.Format("2006-01-02"), status, notesEnc, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update qualification")
		return
	}

	after, _ := a.loadQualificationForAudit(c.Request.Context(), id)
	a.logAuditDetailed(c, user.ID, "compliance", "INFO", "compliance.qualification.update", "qualifications", strconv.FormatInt(id, 10), before, after, req)
	writeSuccess(c, http.StatusOK, gin.H{"id": id})
}

func (a *API) DeleteQualification(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid qualification id")
		return
	}

	before, _ := a.loadQualificationForAudit(c.Request.Context(), id)
	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	res, err := a.db.Exec("DELETE FROM qualifications WHERE id = ? AND "+where, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to delete qualification")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "qualification not found in your scope")
		return
	}

	a.logAuditDetailed(c, user.ID, "compliance", "WARN", "compliance.qualification.delete", "qualifications", strconv.FormatInt(id, 10), before, nil, gin.H{"deleted": true})
	writeSuccess(c, http.StatusOK, gin.H{"deleted": true})
}

type restrictionRequest struct {
	MedName              string  `json:"med_name" binding:"required,min=1,max=128"`
	RuleType             string  `json:"rule_type" binding:"required,min=1,max=64"`
	MaxQuantity          float64 `json:"max_quantity" binding:"required,gt=0"`
	RequiresApproval     bool    `json:"requires_approval"`
	RequiresPrescription bool    `json:"requires_prescription"`
	MinIntervalDays      int     `json:"min_interval_days" binding:"omitempty,gte=1,lte=365"`
	FeeAmount            float64 `json:"fee_amount" binding:"omitempty,gte=0,lte=999999999"`
	FeeCurrency          string  `json:"fee_currency" binding:"omitempty,max=8"`
	IsActive             bool    `json:"is_active"`
}

func normalizeRestrictionRequest(req *restrictionRequest) {
	req.MedName = strings.TrimSpace(req.MedName)
	req.RuleType = strings.TrimSpace(req.RuleType)
	req.FeeCurrency = strings.ToUpper(strings.TrimSpace(req.FeeCurrency))
	if req.MinIntervalDays <= 0 {
		req.MinIntervalDays = 7
	}
	if req.FeeCurrency == "" {
		req.FeeCurrency = "USD"
	}
}

func (a *API) CreateRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req restrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid restriction payload")
		return
	}
	normalizeRestrictionRequest(&req)
	if req.MedName == "" || req.RuleType == "" || req.MaxQuantity <= 0 {
		badRequest(c, "MISSING_REQUIRED_FIELDS", "med_name, rule_type, and max_quantity are required")
		return
	}

	res, err := a.db.Exec(`
		INSERT INTO restrictions
		(med_name, rule_type, max_quantity, requires_approval, requires_prescription, min_interval_days, fee_amount, fee_currency,
		 institution, department, team, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.MedName, req.RuleType, req.MaxQuantity, req.RequiresApproval, req.RequiresPrescription, req.MinIntervalDays, req.FeeAmount, req.FeeCurrency,
		user.Institution, user.Department, user.Team, req.IsActive, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create restriction")
		return
	}
	id, _ := res.LastInsertId()
	after, _ := a.complianceSvc.RestrictionBeforeAfter(c.Request.Context(), id)
	a.logAuditDetailed(c, user.ID, "compliance", "INFO", "compliance.restriction.create", "restrictions", strconv.FormatInt(id, 10), nil, after, req)
	writeSuccess(c, http.StatusCreated, gin.H{"id": id})
}

func (a *API) ListRestrictions(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")

	rows, err := a.db.Query(`
		SELECT id, med_name, rule_type, max_quantity, requires_approval, requires_prescription,
		       min_interval_days, fee_amount, fee_currency, institution, department, team, is_active, created_at, updated_at
		FROM restrictions
		WHERE `+where+`
		ORDER BY id DESC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list restrictions")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id                   int64
			medName              string
			ruleType             string
			maxQuantity          float64
			requiresApproval     bool
			requiresPrescription bool
			minIntervalDays      int
			feeAmount            float64
			feeCurrency          string
			institution          string
			department           string
			team                 string
			isActive             bool
			createdAt            time.Time
			updatedAt            time.Time
		)
		if err := rows.Scan(&id, &medName, &ruleType, &maxQuantity, &requiresApproval, &requiresPrescription, &minIntervalDays, &feeAmount, &feeCurrency,
			&institution, &department, &team, &isActive, &createdAt, &updatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan restrictions")
			return
		}
		items = append(items, gin.H{
			"id":                    id,
			"med_name":              medName,
			"rule_type":             ruleType,
			"max_quantity":          maxQuantity,
			"requires_approval":     requiresApproval,
			"requires_prescription": requiresPrescription,
			"min_interval_days":     minIntervalDays,
			"fee_amount":            feeAmount,
			"fee_currency":          feeCurrency,
			"institution":           institution,
			"department":            department,
			"team":                  team,
			"is_active":             isActive,
			"created_at":            createdAt.Format(time.RFC3339),
			"updated_at":            updatedAt.Format(time.RFC3339),
		})
	}

	writeSuccess(c, http.StatusOK, items)
}

func (a *API) UpdateRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid restriction id")
		return
	}

	var req restrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid restriction payload")
		return
	}
	normalizeRestrictionRequest(&req)

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	var exists int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM restrictions WHERE id = ? AND "+where, args...).Scan(&exists); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate restriction")
		return
	}
	if exists == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "restriction not found in your scope")
		return
	}

	before, _ := a.complianceSvc.RestrictionBeforeAfter(c.Request.Context(), id)
	_, err = a.db.Exec(`
		UPDATE restrictions
		SET med_name = ?, rule_type = ?, max_quantity = ?, requires_approval = ?, requires_prescription = ?,
		    min_interval_days = ?, fee_amount = ?, fee_currency = ?, is_active = ?
		WHERE id = ?
	`, req.MedName, req.RuleType, req.MaxQuantity, req.RequiresApproval, req.RequiresPrescription,
		req.MinIntervalDays, req.FeeAmount, req.FeeCurrency, req.IsActive, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update restriction")
		return
	}

	after, _ := a.complianceSvc.RestrictionBeforeAfter(c.Request.Context(), id)
	a.logAuditDetailed(c, user.ID, "compliance", "INFO", "compliance.restriction.update", "restrictions", strconv.FormatInt(id, 10), before, after, req)
	writeSuccess(c, http.StatusOK, gin.H{"id": id})
}

func (a *API) DeleteRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid restriction id")
		return
	}

	before, _ := a.complianceSvc.RestrictionBeforeAfter(c.Request.Context(), id)
	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	res, err := a.db.Exec("DELETE FROM restrictions WHERE id = ? AND "+where, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to delete restriction")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "restriction not found in your scope")
		return
	}

	a.logAuditDetailed(c, user.ID, "compliance", "WARN", "compliance.restriction.delete", "restrictions", strconv.FormatInt(id, 10), before, nil, gin.H{"deleted": true})
	writeSuccess(c, http.StatusOK, gin.H{"deleted": true})
}

type restrictionCheckRequest struct {
	MedName                  string  `json:"med_name" binding:"required,min=1,max=128"`
	Quantity                 float64 `json:"quantity" binding:"required,gt=0"`
	ClientID                 string  `json:"client_id" binding:"required,min=1,max=64"`
	PrescriptionAttachmentID int64   `json:"prescription_attachment_id" binding:"omitempty,gte=1"`
	RecordPurchase           *bool   `json:"record_purchase"`
}

func (a *API) CheckRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req restrictionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid check payload")
		return
	}
	if strings.TrimSpace(req.MedName) == "" || req.Quantity <= 0 || strings.TrimSpace(req.ClientID) == "" {
		badRequest(c, "INVALID_REQUEST", "med_name, client_id and positive quantity are required")
		return
	}

	recordPurchase := true
	if req.RecordPurchase != nil {
		recordPurchase = *req.RecordPurchase
	}

	result, err := a.complianceSvc.CheckPurchaseRestriction(c.Request.Context(), user, service.PurchaseCheckInput{
		MedName:                  req.MedName,
		Quantity:                 req.Quantity,
		ClientID:                 req.ClientID,
		PrescriptionAttachmentID: req.PrescriptionAttachmentID,
		RecordPurchase:           recordPurchase,
	})
	if err != nil {
		badRequest(c, "CHECK_FAILED", err.Error())
		return
	}

	a.logAuditDetailed(c, user.ID, "compliance", "INFO", "compliance.restriction.check", "restrictions", strconv.FormatInt(result.RuleID, 10), req, result, gin.H{
		"record_purchase": recordPurchase,
	})
	writeSuccess(c, http.StatusOK, result)
}

func (a *API) loadQualificationForAudit(ctx context.Context, id int64) (map[string]any, error) {
	var (
		entityType        string
		entityName        string
		qualificationCode string
		issueDate         time.Time
		expiryDate        time.Time
		status            string
		notesEnc          sql.NullString
	)
	if err := a.db.QueryRowContext(ctx, `
		SELECT entity_type, entity_name, qualification_code, issue_date, expiry_date, status, notes_enc
		FROM qualifications
		WHERE id = ?
	`, id).Scan(&entityType, &entityName, &qualificationCode, &issueDate, &expiryDate, &status, &notesEnc); err != nil {
		return nil, err
	}

	notes := ""
	if notesEnc.Valid && notesEnc.String != "" {
		if raw, err := a.cipher.Decrypt(notesEnc.String); err == nil {
			notes = security.MaskText(raw)
		}
	}

	return map[string]any{
		"entity_type":        entityType,
		"entity_name":        entityName,
		"qualification_code": qualificationCode,
		"issue_date":         issueDate.Format("2006-01-02"),
		"expiry_date":        expiryDate.Format("2006-01-02"),
		"status":             status,
		"notes":              notes,
	}, nil
}

var _ = context.Background
