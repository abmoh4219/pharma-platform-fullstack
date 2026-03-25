package handler

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
)

type createPositionRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=128"`
	Description string `json:"description" binding:"max=2000"`
}

func (a *API) CreatePosition(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)

	var req createPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid position payload")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		badRequest(c, "TITLE_REQUIRED", "position title is required")
		return
	}

	res, err := a.db.Exec(`
		INSERT INTO positions (title, description, institution, department, team, status, created_by)
		VALUES (?, ?, ?, ?, ?, 'open', ?)
	`, strings.TrimSpace(req.Title), strPtr(strings.TrimSpace(req.Description)), user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create position")
		return
	}
	id, _ := res.LastInsertId()

	a.logAudit(c, user.ID, "recruitment.position.create", "positions", strconv.FormatInt(id, 10), req)
	writeSuccess(c, http.StatusCreated, gin.H{"id": id})
}

func (a *API) ListPositions(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")

	rows, err := a.db.Query(`
		SELECT id, title, description, institution, department, team, status, created_at, updated_at
		FROM positions
		WHERE `+where+`
		ORDER BY id DESC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list positions")
		return
	}
	defer rows.Close()

	positions := make([]gin.H, 0)
	for rows.Next() {
		var (
			id          int64
			title       string
			description sql.NullString
			institution string
			department  string
			team        string
			status      string
			createdAt   time.Time
			updatedAt   time.Time
		)
		if err := rows.Scan(&id, &title, &description, &institution, &department, &team, &status, &createdAt, &updatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan positions")
			return
		}
		positions = append(positions, gin.H{
			"id":          id,
			"title":       title,
			"description": description.String,
			"institution": institution,
			"department":  department,
			"team":        team,
			"status":      status,
			"created_at":  createdAt.Format(time.RFC3339),
			"updated_at":  updatedAt.Format(time.RFC3339),
		})
	}

	writeSuccess(c, http.StatusOK, positions)
}

type candidateUpsertRequest struct {
	FullName   string `json:"full_name" binding:"required,min=1,max=128"`
	Phone      string `json:"phone" binding:"required,min=5,max=32"`
	IDNumber   string `json:"id_number" binding:"required,min=3,max=64"`
	Email      string `json:"email" binding:"omitempty,email,max=128"`
	PositionID *int64 `json:"position_id" binding:"omitempty,gte=1"`
	Status     string `json:"status" binding:"omitempty,oneof=new imported shortlisted rejected"`
	ResumePath string `json:"resume_path" binding:"max=255"`
}

var allowedCandidateStatus = map[string]struct{}{
	"new":         {},
	"imported":    {},
	"shortlisted": {},
	"rejected":    {},
}

func (a *API) CreateCandidate(c *gin.Context) {
	a.upsertCandidate(c, 0)
}

func (a *API) UpdateCandidate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(c, "INVALID_ID", "invalid candidate id")
		return
	}
	a.upsertCandidate(c, id)
}

func (a *API) upsertCandidate(c *gin.Context, candidateID int64) {
	user, _ := middleware.GetAuthUser(c)

	var req candidateUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid candidate payload")
		return
	}

	if strings.TrimSpace(req.FullName) == "" || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.IDNumber) == "" {
		badRequest(c, "MISSING_REQUIRED_FIELDS", "full_name, phone and id_number are required")
		return
	}

	phoneEnc, err := a.cipher.Encrypt(strings.TrimSpace(req.Phone))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt phone")
		return
	}
	idEnc, err := a.cipher.Encrypt(strings.TrimSpace(req.IDNumber))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "ENCRYPTION_ERROR", "failed to encrypt id number")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "new"
	}
	if _, ok := allowedCandidateStatus[status]; !ok {
		badRequest(c, "INVALID_STATUS", "status must be one of: new, imported, shortlisted, rejected")
		return
	}

	if candidateID == 0 {
		res, err := a.db.Exec(`
			INSERT INTO candidates
			(full_name, phone_enc, id_number_enc, email, resume_path, position_id, institution, department, team, status, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, strings.TrimSpace(req.FullName), phoneEnc, idEnc, strPtr(strings.TrimSpace(req.Email)), strPtr(strings.TrimSpace(req.ResumePath)), req.PositionID,
			user.Institution, user.Department, user.Team, status, user.ID)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create candidate")
			return
		}
		newID, _ := res.LastInsertId()
		a.logAudit(c, user.ID, "recruitment.candidate.create", "candidates", strconv.FormatInt(newID, 10), gin.H{"full_name": req.FullName})
		writeSuccess(c, http.StatusCreated, gin.H{"id": newID})
		return
	}

	where, scopeArgs := middleware.BuildScopeWhere(user, "")
	args := []any{candidateID}
	args = append(args, scopeArgs...)
	var exists int
	if err := a.db.QueryRow("SELECT COUNT(1) FROM candidates WHERE id = ? AND "+where, args...).Scan(&exists); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate candidate")
		return
	}
	if exists == 0 {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found in your scope")
		return
	}

	_, err = a.db.Exec(`
		UPDATE candidates
		SET full_name = ?, phone_enc = ?, id_number_enc = ?, email = ?, resume_path = ?, position_id = ?, status = ?
		WHERE id = ?
	`, strings.TrimSpace(req.FullName), phoneEnc, idEnc, strPtr(strings.TrimSpace(req.Email)), strPtr(strings.TrimSpace(req.ResumePath)), req.PositionID, status, candidateID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update candidate")
		return
	}

	a.logAudit(c, user.ID, "recruitment.candidate.update", "candidates", strconv.FormatInt(candidateID, 10), gin.H{"full_name": req.FullName})
	writeSuccess(c, http.StatusOK, gin.H{"id": candidateID})
}

func (a *API) ListCandidates(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "c")

	rows, err := a.db.Query(`
		SELECT c.id, c.full_name, c.phone_enc, c.id_number_enc, c.email, c.resume_path, c.position_id, c.status,
		       c.institution, c.department, c.team, c.created_at, c.updated_at, COALESCE(p.title, '')
		FROM candidates c
		LEFT JOIN positions p ON p.id = c.position_id
		WHERE `+where+`
		ORDER BY c.id DESC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list candidates")
		return
	}
	defer rows.Close()

	candidates := make([]gin.H, 0)
	for rows.Next() {
		candidate, err := a.scanCandidateRow(rows)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "DECRYPT_ERROR", "failed to decode candidate")
			return
		}
		candidates = append(candidates, candidate)
	}

	writeSuccess(c, http.StatusOK, candidates)
}

func (a *API) scanCandidateRow(scanner interface{ Scan(dest ...any) error }) (gin.H, error) {
	var (
		id          int64
		fullName    string
		phoneEnc    string
		idEnc       string
		email       sql.NullString
		resumePath  sql.NullString
		positionID  sql.NullInt64
		status      string
		institution string
		department  string
		team        string
		createdAt   time.Time
		updatedAt   time.Time
		position    string
	)
	if err := scanner.Scan(&id, &fullName, &phoneEnc, &idEnc, &email, &resumePath, &positionID, &status, &institution, &department, &team, &createdAt, &updatedAt, &position); err != nil {
		return nil, err
	}

	phoneRaw, err := a.cipher.Decrypt(phoneEnc)
	if err != nil {
		return nil, err
	}
	idRaw, err := a.cipher.Decrypt(idEnc)
	if err != nil {
		return nil, err
	}

	out := gin.H{
		"id":             id,
		"full_name":      fullName,
		"phone":          security.MaskPhone(phoneRaw),
		"id_number":      security.MaskID(idRaw),
		"email":          email.String,
		"resume_path":    resumePath.String,
		"position_id":    positionID.Int64,
		"position_title": position,
		"status":         status,
		"institution":    institution,
		"department":     department,
		"team":           team,
		"created_at":     createdAt.Format(time.RFC3339),
		"updated_at":     updatedAt.Format(time.RFC3339),
	}
	if !positionID.Valid {
		out["position_id"] = nil
	}
	return out, nil
}

type candidateMergeRequest struct {
	PrimaryCandidateID int64   `json:"primary_candidate_id" binding:"required,gte=1"`
	DuplicateIDs       []int64 `json:"duplicate_ids" binding:"required,min=1,dive,gte=1"`
}

func (a *API) MergeCandidates(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req candidateMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid merge payload")
		return
	}
	if req.PrimaryCandidateID <= 0 || len(req.DuplicateIDs) == 0 {
		badRequest(c, "INVALID_MERGE_REQUEST", "primary_candidate_id and duplicate_ids are required")
		return
	}

	primary, err := a.loadCandidateRaw(req.PrimaryCandidateID)
	if err != nil {
		writeError(c, http.StatusNotFound, "PRIMARY_NOT_FOUND", "primary candidate not found")
		return
	}
	if !a.isInScope(user, primary.Institution, primary.Department, primary.Team) {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "candidate is outside your data scope")
		return
	}

	merged := make([]int64, 0)
	for _, duplicateID := range req.DuplicateIDs {
		dup, err := a.loadCandidateRaw(duplicateID)
		if err != nil {
			continue
		}
		if !a.isInScope(user, dup.Institution, dup.Department, dup.Team) {
			continue
		}
		if duplicateID == req.PrimaryCandidateID {
			continue
		}
		if dup.PhoneRaw != primary.PhoneRaw && dup.IDRaw != primary.IDRaw {
			continue
		}
		_, _ = a.db.Exec(`UPDATE attachments SET record_id = ? WHERE module_name = 'candidates' AND record_id = ?`, req.PrimaryCandidateID, duplicateID)
		_, _ = a.db.Exec(`DELETE FROM candidates WHERE id = ?`, duplicateID)
		merged = append(merged, duplicateID)
	}

	a.logAudit(c, user.ID, "recruitment.candidate.merge", "candidates", strconv.FormatInt(req.PrimaryCandidateID, 10), gin.H{
		"merged_ids": merged,
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"primary_candidate_id": req.PrimaryCandidateID,
		"merged_ids":           merged,
	})
}

type candidateRaw struct {
	ID          int64
	FullName    string
	PhoneRaw    string
	IDRaw       string
	Institution string
	Department  string
	Team        string
}

func (a *API) loadCandidateRaw(id int64) (candidateRaw, error) {
	var (
		cand     candidateRaw
		phoneEnc string
		idEnc    string
	)
	err := a.db.QueryRow(`
		SELECT id, full_name, phone_enc, id_number_enc, institution, department, team
		FROM candidates WHERE id = ?
	`, id).Scan(&cand.ID, &cand.FullName, &phoneEnc, &idEnc, &cand.Institution, &cand.Department, &cand.Team)
	if err != nil {
		return candidateRaw{}, err
	}
	cand.PhoneRaw, err = a.cipher.Decrypt(phoneEnc)
	if err != nil {
		return candidateRaw{}, err
	}
	cand.IDRaw, err = a.cipher.Decrypt(idEnc)
	if err != nil {
		return candidateRaw{}, err
	}
	return cand, nil
}

func (a *API) isInScope(user middleware.AuthUser, institution, department, team string) bool {
	if user.Role == "system_admin" {
		return true
	}
	return user.Institution == institution && user.Department == department && user.Team == team
}

type smartSearchResult struct {
	CandidateID int64    `json:"candidate_id"`
	FullName    string   `json:"full_name"`
	MaskedPhone string   `json:"masked_phone"`
	MaskedID    string   `json:"masked_id"`
	Score       int      `json:"score"`
	Explanation []string `json:"explanation"`
	Institution string   `json:"institution"`
	Department  string   `json:"department"`
	Team        string   `json:"team"`
}

func (a *API) SmartSearchCandidates(c *gin.Context) {
	var req struct {
		Query string `form:"q" binding:"required,min=1,max=128"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		badRequest(c, "QUERY_REQUIRED", "q is required")
		return
	}

	query := strings.TrimSpace(strings.ToLower(req.Query))
	if query == "" {
		badRequest(c, "QUERY_REQUIRED", "q is required")
		return
	}

	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")
	rows, err := a.db.Query(`
		SELECT id, full_name, phone_enc, id_number_enc, COALESCE(email, ''), institution, department, team
		FROM candidates
		WHERE `+where+`
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to query candidates")
		return
	}
	defer rows.Close()

	tokens := strings.Fields(query)
	results := make([]smartSearchResult, 0)
	for rows.Next() {
		var (
			id          int64
			fullName    string
			phoneEnc    string
			idEnc       string
			email       string
			institution string
			department  string
			team        string
		)
		if err := rows.Scan(&id, &fullName, &phoneEnc, &idEnc, &email, &institution, &department, &team); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan candidate")
			return
		}

		phoneRaw, err := a.cipher.Decrypt(phoneEnc)
		if err != nil {
			continue
		}
		idRaw, err := a.cipher.Decrypt(idEnc)
		if err != nil {
			continue
		}

		score, explanation := ScoreCandidate(tokens, fullName, email, phoneRaw, idRaw)
		if score == 0 {
			continue
		}
		results = append(results, smartSearchResult{
			CandidateID: id,
			FullName:    fullName,
			MaskedPhone: security.MaskPhone(phoneRaw),
			MaskedID:    security.MaskID(idRaw),
			Score:       score,
			Explanation: explanation,
			Institution: institution,
			Department:  department,
			Team:        team,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].CandidateID > results[j].CandidateID
		}
		return results[i].Score > results[j].Score
	})

	writeSuccess(c, http.StatusOK, results)
}

func ScoreCandidate(tokens []string, fullName, email, phone, idNumber string) (int, []string) {
	score := 0
	explanation := make([]string, 0)
	nameLower := strings.ToLower(fullName)
	emailLower := strings.ToLower(email)

	for _, t := range tokens {
		if strings.Contains(nameLower, t) {
			score += 35
			explanation = append(explanation, fmt.Sprintf("name matched '%s'", t))
		}
		if emailLower != "" && strings.Contains(emailLower, t) {
			score += 15
			explanation = append(explanation, fmt.Sprintf("email matched '%s'", t))
		}
		if strings.Contains(strings.ToLower(phone), t) {
			score += 25
			explanation = append(explanation, fmt.Sprintf("phone matched '%s'", t))
		}
		if strings.Contains(strings.ToLower(idNumber), t) {
			score += 30
			explanation = append(explanation, fmt.Sprintf("id_number matched '%s'", t))
		}
	}

	if score > 100 {
		score = 100
	}
	return score, explanation
}

func (a *API) ImportCandidates(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "FILE_REQUIRED", "file is required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "FILE_OPEN_ERROR", "failed to open uploaded file")
		return
	}
	defer file.Close()

	rows, err := parseCandidateImportRows(file, fileHeader)
	if err != nil {
		badRequest(c, "IMPORT_PARSE_ERROR", "failed to parse import file")
		return
	}

	imported := 0
	failed := make([]gin.H, 0)
	for i, row := range rows {
		fullName := strings.TrimSpace(row["full_name"])
		phone := strings.TrimSpace(row["phone"])
		idNumber := strings.TrimSpace(row["id_number"])
		email := strings.TrimSpace(row["email"])
		if fullName == "" || phone == "" || idNumber == "" {
			failed = append(failed, gin.H{"row": i + 2, "reason": "missing full_name/phone/id_number"})
			continue
		}
		if err := a.insertImportedCandidate(user, fullName, phone, idNumber, email); err != nil {
			failed = append(failed, gin.H{"row": i + 2, "reason": err.Error()})
			continue
		}
		imported++
	}

	a.logAudit(c, user.ID, "recruitment.candidate.import", "candidates", "bulk", gin.H{
		"file":     fileHeader.Filename,
		"imported": imported,
		"failed":   len(failed),
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"total_rows": len(rows),
		"imported":   imported,
		"failed":     failed,
	})
}

func (a *API) insertImportedCandidate(user middleware.AuthUser, fullName, phone, idNumber, email string) error {
	phoneEnc, err := a.cipher.Encrypt(phone)
	if err != nil {
		return fmt.Errorf("encrypt phone")
	}
	idEnc, err := a.cipher.Encrypt(idNumber)
	if err != nil {
		return fmt.Errorf("encrypt id")
	}

	_, err = a.db.Exec(`
		INSERT INTO candidates
		(full_name, phone_enc, id_number_enc, email, institution, department, team, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'imported', ?)
	`, fullName, phoneEnc, idEnc, strPtr(email), user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		return fmt.Errorf("unable to save candidate row")
	}
	return nil
}

func parseCandidateImportRows(file multipart.File, fileHeader *multipart.FileHeader) ([]map[string]string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	switch ext {
	case ".csv":
		return parseCSVRows(file)
	case ".xlsx", ".xls":
		return parseExcelRows(file)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func parseCSVRows(file multipart.File) ([]map[string]string, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty csv file")
	}
	headers := normalizeHeaders(records[0])
	rows := make([]map[string]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(map[string]string)
		for i, h := range headers {
			if i < len(rec) {
				row[h] = strings.TrimSpace(rec[i])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseExcelRows(file multipart.File) ([]map[string]string, error) {
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp("", "import-*.xlsx")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(buf); err != nil {
		_ = tmpFile.Close()
		return nil, err
	}
	_ = tmpFile.Close()

	xf, err := excelize.OpenFile(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	defer func() { _ = xf.Close() }()

	sheet := xf.GetSheetName(0)
	if sheet == "" {
		return nil, fmt.Errorf("excel has no sheet")
	}
	records, err := xf.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty excel file")
	}

	headers := normalizeHeaders(records[0])
	rows := make([]map[string]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(map[string]string)
		for i, h := range headers {
			if i < len(rec) {
				row[h] = strings.TrimSpace(rec[i])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func normalizeHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		n := strings.ToLower(strings.TrimSpace(h))
		n = strings.ReplaceAll(n, " ", "_")
		switch n {
		case "name":
			n = "full_name"
		case "id", "id_no", "idnumber":
			n = "id_number"
		}
		out[i] = n
	}
	return out
}
