package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
)

type caseCreateRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

func (a *API) CreateCase(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req caseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid case payload")
		return
	}
	subject := strings.TrimSpace(req.Subject)
	description := strings.TrimSpace(req.Description)
	if subject == "" || description == "" {
		badRequest(c, "MISSING_REQUIRED_FIELDS", "subject and description are required")
		return
	}

	var recentCount int
	if err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM case_ledgers
		WHERE subject = ? AND institution = ? AND department = ? AND team = ?
		AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 5 MINUTE)
	`, subject, user.Institution, user.Department, user.Team).Scan(&recentCount); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate duplicate cases")
		return
	}
	if recentCount > 0 {
		writeError(c, http.StatusConflict, "DUPLICATE_BLOCK", "a similar case was created in the last 5 minutes")
		return
	}

	caseNo, err := a.generateCaseNumber(user.Institution)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "CASE_NO_GENERATION_FAILED", "failed to generate case number")
		return
	}
	descriptionEnc, err := a.cipher.Encrypt(description)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt case description")
		return
	}

	res, err := a.db.Exec(`
		INSERT INTO case_ledgers
		(case_no, subject, description_enc, institution, department, team, status, assigned_to, created_by, last_transition_at)
		VALUES (?, ?, ?, ?, ?, ?, 'new', NULL, ?, UTC_TIMESTAMP())
	`, caseNo, subject, descriptionEnc, user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create case")
		return
	}
	id, _ := res.LastInsertId()

	a.logAudit(c, user.ID, "cases.create", "case_ledgers", strconv.FormatInt(id, 10), gin.H{"case_no": caseNo, "subject": subject})
	writeSuccess(c, http.StatusCreated, gin.H{"id": id, "case_no": caseNo})
}

func (a *API) generateCaseNumber(institution string) (string, error) {
	prefix := time.Now().UTC().Format("20060102") + "-" + NormalizeInstitutionPart(institution) + "-"
	likePrefix := prefix + "%"
	var count int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM case_ledgers WHERE case_no LIKE ?", likePrefix).Scan(&count); err != nil {
		return "", err
	}
	return FormatCaseNumber(time.Now().UTC(), institution, count+1), nil
}

func FormatCaseNumber(at time.Time, institution string, sequence int) string {
	return at.UTC().Format("20060102") + "-" + NormalizeInstitutionPart(institution) + "-" + leftPadInt(sequence, 6)
}

func leftPadInt(v int, width int) string {
	text := strconv.Itoa(v)
	if len(text) >= width {
		return text
	}
	return strings.Repeat("0", width-len(text)) + text
}

func (a *API) ListCases(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	statusFilter := strings.TrimSpace(c.Query("status"))
	q := strings.TrimSpace(strings.ToLower(c.Query("q")))

	where, args := middleware.BuildScopeWhere(user, "cl")
	if statusFilter != "" {
		where += " AND cl.status = ?"
		args = append(args, statusFilter)
	}
	if q != "" {
		where += " AND (LOWER(cl.case_no) LIKE ? OR LOWER(cl.subject) LIKE ?)"
		like := "%" + q + "%"
		args = append(args, like, like)
	}

	rows, err := a.db.Query(`
		SELECT cl.id, cl.case_no, cl.subject, cl.description_enc, cl.status, cl.assigned_to,
		       cl.institution, cl.department, cl.team, cl.created_by, cl.last_transition_at, cl.created_at, cl.updated_at,
		       COALESCE(u.full_name, '')
		FROM case_ledgers cl
		LEFT JOIN users u ON u.id = cl.assigned_to
		WHERE `+where+`
		ORDER BY cl.id DESC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list cases")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id               int64
			caseNo           string
			subject          string
			descriptionEnc   string
			status           string
			assignedTo       sql.NullInt64
			institution      string
			department       string
			team             string
			createdBy        int64
			lastTransitionAt time.Time
			createdAt        time.Time
			updatedAt        time.Time
			assignedName     string
		)
		if err := rows.Scan(&id, &caseNo, &subject, &descriptionEnc, &status, &assignedTo, &institution, &department, &team, &createdBy, &lastTransitionAt, &createdAt, &updatedAt, &assignedName); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan cases")
			return
		}
		descriptionRaw, err := a.cipher.Decrypt(descriptionEnc)
		if err != nil {
			descriptionRaw = ""
		}

		item := gin.H{
			"id":                 id,
			"case_no":            caseNo,
			"subject":            subject,
			"description":        security.MaskText(descriptionRaw),
			"status":             status,
			"assigned_to":        nil,
			"assigned_to_name":   assignedName,
			"institution":        institution,
			"department":         department,
			"team":               team,
			"created_by":         createdBy,
			"last_transition_at": lastTransitionAt.Format(time.RFC3339),
			"created_at":         createdAt.Format(time.RFC3339),
			"updated_at":         updatedAt.Format(time.RFC3339),
		}
		if assignedTo.Valid {
			item["assigned_to"] = assignedTo.Int64
		}
		items = append(items, item)
	}

	writeSuccess(c, http.StatusOK, items)
}

type assignCaseRequest struct {
	AssignedTo int64 `json:"assigned_to"`
}

func (a *API) AssignCase(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid case id")
		return
	}

	var req assignCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid assignment payload")
		return
	}
	if req.AssignedTo <= 0 {
		badRequest(c, "INVALID_ASSIGNEE", "assigned_to must be a valid user id")
		return
	}

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	var exists int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM case_ledgers WHERE id = ? AND "+where, args...).Scan(&exists); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate case")
		return
	}
	if exists == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "case not found in your scope")
		return
	}

	_, err = a.db.Exec(`
		UPDATE case_ledgers
		SET assigned_to = ?, status = 'assigned', last_transition_at = UTC_TIMESTAMP()
		WHERE id = ?
	`, req.AssignedTo, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to assign case")
		return
	}

	a.logAudit(c, user.ID, "cases.assign", "case_ledgers", strconv.FormatInt(id, 10), req)
	writeSuccess(c, http.StatusOK, gin.H{"id": id, "assigned_to": req.AssignedTo})
}

type caseStatusRequest struct {
	Status string `json:"status"`
}

var allowedTransitions = map[string]map[string]struct{}{
	"new": {
		"assigned":    {},
		"in_progress": {},
	},
	"assigned": {
		"in_progress": {},
		"resolved":    {},
	},
	"in_progress": {
		"resolved": {},
	},
	"resolved": {
		"closed": {},
	},
	"closed": {},
}

func (a *API) UpdateCaseStatus(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid case id")
		return
	}

	var req caseStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid status payload")
		return
	}
	newStatus := strings.TrimSpace(req.Status)
	if newStatus == "" {
		badRequest(c, "INVALID_STATUS", "status is required")
		return
	}

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	var currentStatus string
	if err := a.db.QueryRow("SELECT status FROM case_ledgers WHERE id = ? AND "+where, args...).Scan(&currentStatus); err != nil {
		if err == sql.ErrNoRows {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "case not found in your scope")
			return
		}
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to read current case status")
		return
	}

	if _, ok := allowedTransitions[currentStatus][newStatus]; !ok {
		writeError(c, http.StatusConflict, "INVALID_STATUS_TRANSITION", "invalid status transition")
		return
	}

	_, err = a.db.Exec(`
		UPDATE case_ledgers
		SET status = ?, last_transition_at = UTC_TIMESTAMP()
		WHERE id = ?
	`, newStatus, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update case status")
		return
	}

	a.logAudit(c, user.ID, "cases.status.update", "case_ledgers", strconv.FormatInt(id, 10), gin.H{"from": currentStatus, "to": newStatus})
	writeSuccess(c, http.StatusOK, gin.H{"id": id, "status": newStatus})
}

func (a *API) ListCaseAttachments(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid case id")
		return
	}

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{id}
	args = append(args, scopeArgs...)
	var exists int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM case_ledgers WHERE id = ? AND "+where, args...).Scan(&exists); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate case")
		return
	}
	if exists == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "case not found in your scope")
		return
	}

	rows, err := a.db.Query(`
		SELECT id, original_name, file_path, mime_type, file_size, sha256, created_at
		FROM attachments
		WHERE module_name = 'case_ledgers' AND record_id = ?
		ORDER BY id DESC
	`, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list attachments")
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			attachmentID int64
			originalName string
			filePath     string
			mimeType     string
			fileSize     int64
			sha256       string
			createdAt    time.Time
		)
		if err := rows.Scan(&attachmentID, &originalName, &filePath, &mimeType, &fileSize, &sha256, &createdAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan attachments")
			return
		}
		items = append(items, gin.H{
			"id":            attachmentID,
			"original_name": originalName,
			"file_path":     filePath,
			"mime_type":     mimeType,
			"file_size":     fileSize,
			"sha256":        sha256,
			"created_at":    createdAt.Format(time.RFC3339),
		})
	}

	writeSuccess(c, http.StatusOK, items)
}
