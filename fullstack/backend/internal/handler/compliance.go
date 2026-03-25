package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
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
	a.logAudit(c, user.ID, "compliance.qualification.create", "qualifications", strconv.FormatInt(id, 10), req)
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
	if req.EntityType != "client" && req.EntityType != "supplier" {
		badRequest(c, "INVALID_ENTITY_TYPE", "entity_type must be client or supplier")
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
	notesEnc, err := a.cipher.Encrypt(strings.TrimSpace(req.Notes))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt notes")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
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

	_, err = a.db.Exec(`
		UPDATE qualifications
		SET entity_type = ?, entity_name = ?, qualification_code = ?, issue_date = ?, expiry_date = ?, status = ?, notes_enc = ?
		WHERE id = ?
	`, req.EntityType, strings.TrimSpace(req.EntityName), strings.TrimSpace(req.QualificationCode), issueDate.Format("2006-01-02"), expiryDate.Format("2006-01-02"), status, notesEnc, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update qualification")
		return
	}

	a.logAudit(c, user.ID, "compliance.qualification.update", "qualifications", strconv.FormatInt(id, 10), req)
	writeSuccess(c, http.StatusOK, gin.H{"id": id})
}

func (a *API) DeleteQualification(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid qualification id")
		return
	}
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

	a.logAudit(c, user.ID, "compliance.qualification.delete", "qualifications", strconv.FormatInt(id, 10), gin.H{})
	writeSuccess(c, http.StatusOK, gin.H{"deleted": true})
}

type restrictionRequest struct {
	MedName          string  `json:"med_name" binding:"required,min=1,max=128"`
	RuleType         string  `json:"rule_type" binding:"required,min=1,max=64"`
	MaxQuantity      float64 `json:"max_quantity" binding:"required,gt=0"`
	RequiresApproval bool    `json:"requires_approval"`
	IsActive         bool    `json:"is_active"`
}

func (a *API) CreateRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req restrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid restriction payload")
		return
	}
	if strings.TrimSpace(req.MedName) == "" || strings.TrimSpace(req.RuleType) == "" || req.MaxQuantity <= 0 {
		badRequest(c, "MISSING_REQUIRED_FIELDS", "med_name, rule_type, and max_quantity are required")
		return
	}

	res, err := a.db.Exec(`
		INSERT INTO restrictions
		(med_name, rule_type, max_quantity, requires_approval, institution, department, team, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, strings.TrimSpace(req.MedName), strings.TrimSpace(req.RuleType), req.MaxQuantity, req.RequiresApproval,
		user.Institution, user.Department, user.Team, req.IsActive, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create restriction")
		return
	}
	id, _ := res.LastInsertId()
	a.logAudit(c, user.ID, "compliance.restriction.create", "restrictions", strconv.FormatInt(id, 10), req)
	writeSuccess(c, http.StatusCreated, gin.H{"id": id})
}

func (a *API) ListRestrictions(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")

	rows, err := a.db.Query(`
		SELECT id, med_name, rule_type, max_quantity, requires_approval, institution, department, team, is_active, created_at, updated_at
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
			id               int64
			medName          string
			ruleType         string
			maxQuantity      float64
			requiresApproval bool
			institution      string
			department       string
			team             string
			isActive         bool
			createdAt        time.Time
			updatedAt        time.Time
		)
		if err := rows.Scan(&id, &medName, &ruleType, &maxQuantity, &requiresApproval, &institution, &department, &team, &isActive, &createdAt, &updatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan restrictions")
			return
		}
		items = append(items, gin.H{
			"id":                id,
			"med_name":          medName,
			"rule_type":         ruleType,
			"max_quantity":      maxQuantity,
			"requires_approval": requiresApproval,
			"institution":       institution,
			"department":        department,
			"team":              team,
			"is_active":         isActive,
			"created_at":        createdAt.Format(time.RFC3339),
			"updated_at":        updatedAt.Format(time.RFC3339),
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

	_, err = a.db.Exec(`
		UPDATE restrictions
		SET med_name = ?, rule_type = ?, max_quantity = ?, requires_approval = ?, is_active = ?
		WHERE id = ?
	`, strings.TrimSpace(req.MedName), strings.TrimSpace(req.RuleType), req.MaxQuantity, req.RequiresApproval, req.IsActive, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update restriction")
		return
	}

	a.logAudit(c, user.ID, "compliance.restriction.update", "restrictions", strconv.FormatInt(id, 10), req)
	writeSuccess(c, http.StatusOK, gin.H{"id": id})
}

func (a *API) DeleteRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid restriction id")
		return
	}
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

	a.logAudit(c, user.ID, "compliance.restriction.delete", "restrictions", strconv.FormatInt(id, 10), gin.H{})
	writeSuccess(c, http.StatusOK, gin.H{"deleted": true})
}

type restrictionCheckRequest struct {
	MedName  string  `json:"med_name" binding:"required,min=1,max=128"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
}

func (a *API) CheckRestriction(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req restrictionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid check payload")
		return
	}
	if strings.TrimSpace(req.MedName) == "" || req.Quantity <= 0 {
		badRequest(c, "INVALID_REQUEST", "med_name and positive quantity are required")
		return
	}

	where, args := middleware.BuildScopeWhere(user, "")
	args = append([]any{strings.TrimSpace(req.MedName)}, args...)
	query := `
		SELECT id, rule_type, max_quantity, requires_approval, is_active
		FROM restrictions
		WHERE med_name = ? AND ` + where + `
		ORDER BY id DESC LIMIT 1`

	var (
		ruleID           int64
		ruleType         string
		maxQuantity      float64
		requiresApproval bool
		isActive         bool
	)
	if err := a.db.QueryRow(query, args...).Scan(&ruleID, &ruleType, &maxQuantity, &requiresApproval, &isActive); err != nil {
		if err == sql.ErrNoRows {
			writeSuccess(c, http.StatusOK, gin.H{"allowed": true, "reason": "no active rule"})
			return
		}
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to evaluate restriction")
		return
	}

	allowed := true
	reason := "within rule"
	if !isActive {
		reason = "rule is inactive"
	} else if req.Quantity > maxQuantity {
		allowed = false
		reason = fmt.Sprintf("quantity %.2f exceeds max %.2f", req.Quantity, maxQuantity)
	} else if requiresApproval {
		reason = "requires approval"
	}

	a.logAudit(c, user.ID, "compliance.restriction.check", "restrictions", strconv.FormatInt(ruleID, 10), gin.H{
		"med_name":          req.MedName,
		"quantity":          req.Quantity,
		"allowed":           allowed,
		"requires_approval": requiresApproval,
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"rule_id":           ruleID,
		"rule_type":         ruleType,
		"max_quantity":      maxQuantity,
		"requires_approval": requiresApproval,
		"allowed":           allowed,
		"reason":            reason,
	})
}
