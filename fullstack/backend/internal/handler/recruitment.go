package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
	"pharma-platform/internal/service"
)

type createPositionRequest struct {
	Title                string   `json:"title" binding:"required,min=1,max=128"`
	Description          string   `json:"description" binding:"max=2000"`
	RequiredSkills       []string `json:"required_skills"`
	RequiredEducation    string   `json:"required_education_level" binding:"omitempty,max=64"`
	MinYearsExperience   float64  `json:"min_years_experience" binding:"omitempty,gte=0,lte=60"`
	TargetTimeToFillDays int      `json:"target_time_to_fill_days" binding:"omitempty,gte=1,lte=365"`
	Tags                 []string `json:"tags"`
}

func (a *API) CreatePosition(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	var req createPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid position payload")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		badRequest(c, "TITLE_REQUIRED", "position title is required")
		return
	}

	tags := service.NormalizeTags(req.Tags)
	tagsJSON, _ := json.Marshal(tags)
	requiredSkills := strings.Join(service.NormalizeSkills(req.RequiredSkills), ",")
	targetDays := req.TargetTimeToFillDays
	if targetDays <= 0 {
		targetDays = 30
	}

	res, err := a.db.Exec(`
		INSERT INTO positions
		(title, description, required_skills_text, required_education_level, min_years_experience, target_time_to_fill_days, tags_json,
		 institution, department, team, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?)
	`, title, strPtr(strings.TrimSpace(req.Description)), strPtr(requiredSkills), strPtr(strings.TrimSpace(req.RequiredEducation)),
		req.MinYearsExperience, targetDays, string(tagsJSON), user.Institution, user.Department, user.Team, user.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create position")
		return
	}
	id, _ := res.LastInsertId()

	a.logAuditDetailed(c, user.ID, "recruitment", "INFO", "recruitment.position.create", "positions", strconv.FormatInt(id, 10), nil, req, req)
	writeSuccess(c, http.StatusCreated, gin.H{"id": id})
}

func (a *API) ListPositions(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	where, args := middleware.BuildScopeWhere(user, "")
	rows, err := a.db.Query(`
		SELECT id, title, COALESCE(description, ''), COALESCE(required_skills_text, ''), COALESCE(required_education_level, ''),
		       COALESCE(min_years_experience, 0), COALESCE(target_time_to_fill_days, 30), COALESCE(CAST(tags_json AS CHAR), '[]'),
		       institution, department, team, status, created_at, updated_at
		FROM positions
		WHERE `+where+`
		ORDER BY id DESC
	`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list positions")
		return
	}
	defer rows.Close()

	out := make([]gin.H, 0)
	for rows.Next() {
		var (
			id                 int64
			title              string
			description        string
			requiredSkillsText string
			requiredEducation  string
			minYearsExp        float64
			targetDays         int
			tagsRaw            string
			institution        string
			department         string
			team               string
			status             string
			createdAt          time.Time
			updatedAt          time.Time
		)
		if err := rows.Scan(&id, &title, &description, &requiredSkillsText, &requiredEducation, &minYearsExp, &targetDays, &tagsRaw,
			&institution, &department, &team, &status, &createdAt, &updatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to scan positions")
			return
		}
		tags := []string{}
		_ = json.Unmarshal([]byte(tagsRaw), &tags)
		out = append(out, gin.H{
			"id":                       id,
			"title":                    title,
			"description":              description,
			"required_skills":          service.ParseListCSV(requiredSkillsText),
			"required_education_level": requiredEducation,
			"min_years_experience":     minYearsExp,
			"target_time_to_fill_days": targetDays,
			"tags":                     tags,
			"institution":              institution,
			"department":               department,
			"team":                     team,
			"status":                   status,
			"created_at":               createdAt.Format(time.RFC3339),
			"updated_at":               updatedAt.Format(time.RFC3339),
		})
	}

	writeSuccess(c, http.StatusOK, out)
}

type candidateUpsertRequest struct {
	FullName        string            `json:"full_name" binding:"required,min=1,max=128"`
	Phone           string            `json:"phone" binding:"required,min=5,max=32"`
	IDNumber        string            `json:"id_number" binding:"required,min=3,max=64"`
	Email           string            `json:"email" binding:"omitempty,email,max=128"`
	PositionID      *int64            `json:"position_id" binding:"omitempty,gte=1"`
	Status          string            `json:"status" binding:"omitempty,oneof=new imported shortlisted rejected"`
	ResumePath      string            `json:"resume_path" binding:"max=255"`
	Tags            []string          `json:"tags"`
	CustomFields    map[string]string `json:"custom_fields"`
	Skills          []string          `json:"skills"`
	EducationLevel  string            `json:"education_level" binding:"omitempty,max=64"`
	YearsExperience float64           `json:"years_experience" binding:"omitempty,gte=0,lte=60"`
	LastActiveAt    string            `json:"last_active_at" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
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

	var lastActive *time.Time
	if strings.TrimSpace(req.LastActiveAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.LastActiveAt))
		if err != nil {
			badRequest(c, "INVALID_LAST_ACTIVE_AT", "last_active_at must be RFC3339")
			return
		}
		lastActive = &parsed
	}

	before := map[string]any(nil)
	if candidateID > 0 {
		current, err := a.recruitSvc.LoadCandidate(c.Request.Context(), candidateID)
		if err != nil {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if !isInScope(user, current.Institution, current.Department, current.Team) {
			writeError(c, http.StatusForbidden, "FORBIDDEN", "candidate is outside your data scope")
			return
		}
		before = candidateToResponse(current)
	}

	result, err := a.recruitSvc.UpsertCandidate(c.Request.Context(), user, service.CandidateUpsertInput{
		ID:              candidateID,
		FullName:        req.FullName,
		Phone:           req.Phone,
		IDNumber:        req.IDNumber,
		Email:           req.Email,
		ResumePath:      req.ResumePath,
		PositionID:      req.PositionID,
		Status:          req.Status,
		Tags:            req.Tags,
		CustomFields:    req.CustomFields,
		Skills:          req.Skills,
		EducationLevel:  req.EducationLevel,
		YearsExperience: req.YearsExperience,
		LastActiveAt:    lastActive,
	})
	if err != nil {
		badRequest(c, "INVALID_CANDIDATE", err.Error())
		return
	}

	action := "recruitment.candidate.create"
	statusCode := http.StatusCreated
	if candidateID > 0 || result.WasMerged {
		action = "recruitment.candidate.update"
		statusCode = http.StatusOK
	}
	after := candidateToResponse(result.Model)
	details := gin.H{"was_merged": result.WasMerged}
	if result.MergedFrom != nil {
		details["merged_from_id"] = *result.MergedFrom
	}

	a.logAuditDetailed(c, user.ID, "recruitment", "INFO", action, "candidates", strconv.FormatInt(result.Model.ID, 10), before, after, details)
	writeSuccess(c, statusCode, gin.H{"id": result.Model.ID, "was_merged": result.WasMerged})
}

func (a *API) ListCandidates(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	items, err := a.recruitSvc.ListCandidates(c.Request.Context(), user)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list candidates")
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, candidateToResponse(item))
	}
	writeSuccess(c, http.StatusOK, out)
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
	merged, err := a.recruitSvc.MergeDuplicates(c.Request.Context(), user, req.PrimaryCandidateID, req.DuplicateIDs)
	if err != nil {
		writeError(c, http.StatusBadRequest, "MERGE_FAILED", err.Error())
		return
	}

	a.logAuditDetailed(c, user.ID, "recruitment", "INFO", "recruitment.candidate.merge", "candidates", strconv.FormatInt(req.PrimaryCandidateID, 10), nil, gin.H{"merged_ids": merged}, gin.H{"merged_ids": merged})
	writeSuccess(c, http.StatusOK, gin.H{"primary_candidate_id": req.PrimaryCandidateID, "merged_ids": merged})
}

func (a *API) SmartSearchCandidates(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		badRequest(c, "QUERY_REQUIRED", "q is required")
		return
	}
	user, _ := middleware.GetAuthUser(c)
	results, err := a.recruitSvc.SmartSearch(c.Request.Context(), user, query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to search candidates")
		return
	}
	out := make([]gin.H, 0, len(results))
	for _, item := range results {
		out = append(out, gin.H{
			"candidate_id": item.CandidateID,
			"full_name":    item.FullName,
			"masked_phone": item.MaskedPhone,
			"masked_id":    item.MaskedID,
			"score":        item.Score,
			"explanation":  item.Reasons,
			"institution":  item.Institution,
			"department":   item.Department,
			"team":         item.Team,
		})
	}
	writeSuccess(c, http.StatusOK, out)
}

func (a *API) CandidateMatchScore(c *gin.Context) {
	candidateID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || candidateID <= 0 {
		badRequest(c, "INVALID_ID", "invalid candidate id")
		return
	}
	positionID, err := strconv.ParseInt(c.Query("position_id"), 10, 64)
	if err != nil || positionID <= 0 {
		badRequest(c, "INVALID_POSITION_ID", "position_id query param is required")
		return
	}
	user, _ := middleware.GetAuthUser(c)
	result, err := a.recruitSvc.ExplainableMatch(c.Request.Context(), user, candidateID, positionID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "outside your data scope") {
			writeError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		writeError(c, http.StatusBadRequest, "MATCH_SCORE_FAILED", err.Error())
		return
	}
	writeSuccess(c, http.StatusOK, result)
}

func (a *API) CandidateRecommendations(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	candidateID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || candidateID <= 0 {
		badRequest(c, "INVALID_ID", "invalid candidate id")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if limit <= 0 {
		limit = 5
	}

	similarCandidates, err := a.recruitSvc.SimilarCandidates(c.Request.Context(), user, candidateID, limit)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "outside your data scope") {
			writeError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, "RECOMMENDATION_FAILED", err.Error())
		return
	}
	similarPositions, err := a.recruitSvc.SimilarPositions(c.Request.Context(), user, candidateID, limit)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "outside your data scope") {
			writeError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		writeError(c, http.StatusInternalServerError, "RECOMMENDATION_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, gin.H{
		"similar_candidates": similarCandidates,
		"similar_positions":  similarPositions,
	})
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

	imported, failed, err := a.recruitSvc.ImportCandidateRows(c.Request.Context(), user, rows)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "IMPORT_FAILED", err.Error())
		return
	}

	a.logAuditDetailed(c, user.ID, "recruitment", "INFO", "recruitment.candidate.import", "candidates", "bulk", nil, gin.H{"imported": imported, "failed": len(failed)}, gin.H{
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

func candidateToResponse(item service.CandidateModel) gin.H {
	lastActive := ""
	if !item.LastActiveAt.IsZero() {
		lastActive = item.LastActiveAt.UTC().Format(time.RFC3339)
	}

	resp := gin.H{
		"id":               item.ID,
		"full_name":        item.FullName,
		"phone":            security.MaskPhone(item.Phone),
		"id_number":        security.MaskID(item.IDNumber),
		"email":            item.Email,
		"resume_path":      item.ResumePath,
		"position_id":      nil,
		"position_title":   item.PositionTitle,
		"status":           item.Status,
		"tags":             item.Tags,
		"custom_fields":    item.CustomFields,
		"skills":           item.Skills,
		"education_level":  item.EducationLevel,
		"years_experience": item.YearsExperience,
		"last_active_at":   lastActive,
		"institution":      item.Institution,
		"department":       item.Department,
		"team":             item.Team,
		"created_at":       item.CreatedAt.Format(time.RFC3339),
		"updated_at":       item.UpdatedAt.Format(time.RFC3339),
	}
	if item.PositionID != nil {
		resp["position_id"] = *item.PositionID
	}
	return resp
}

func isInScope(user middleware.AuthUser, institution, department, team string) bool {
	if user.Role == "system_admin" {
		return true
	}
	return user.Institution == institution && user.Department == department && user.Team == team
}
