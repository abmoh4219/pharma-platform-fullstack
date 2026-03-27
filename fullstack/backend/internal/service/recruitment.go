package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
)

type RecruitmentService struct {
	db     *sql.DB
	cipher *security.FieldCipher
}

func NewRecruitmentService(db *sql.DB, cipher *security.FieldCipher) *RecruitmentService {
	return &RecruitmentService{db: db, cipher: cipher}
}

type CandidateUpsertInput struct {
	ID              int64
	FullName        string
	Phone           string
	IDNumber        string
	Email           string
	ResumePath      string
	PositionID      *int64
	Status          string
	Tags            []string
	CustomFields    map[string]string
	Skills          []string
	EducationLevel  string
	YearsExperience float64
	LastActiveAt    *time.Time
}

type CandidateModel struct {
	ID              int64
	FullName        string
	Phone           string
	IDNumber        string
	Email           string
	ResumePath      string
	PositionID      *int64
	PositionTitle   string
	Status          string
	Tags            []string
	CustomFields    map[string]string
	Skills          []string
	EducationLevel  string
	YearsExperience float64
	LastActiveAt    time.Time
	Institution     string
	Department      string
	Team            string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type SearchCandidate struct {
	CandidateID int64
	FullName    string
	MaskedPhone string
	MaskedID    string
	Score       int
	Reasons     []string
	Institution string
	Department  string
	Team        string
}

type SimilarCandidate struct {
	CandidateID   int64
	FullName      string
	PositionTitle string
	Similarity    int
	Reasons       []string
}

type SimilarPosition struct {
	PositionID int64
	Title      string
	Similarity int
	Reasons    []string
}

type PositionModel struct {
	ID                   int64
	Title                string
	Description          string
	RequiredSkills       []string
	RequiredEducation    string
	MinYearsExperience   float64
	TargetTimeToFillDays int
	Tags                 []string
	Institution          string
	Department           string
	Team                 string
}

type MatchScoreBreakdown struct {
	Score           int      `json:"score"`
	Weighted        []string `json:"weighted"`
	Reasons         []string `json:"reasons"`
	SkillScore      int      `json:"skill_score"`
	EducationScore  int      `json:"education_score"`
	ExperienceScore int      `json:"experience_score"`
	TimeScore       int      `json:"time_score"`
}

type DuplicateCheckResult struct {
	Model      CandidateModel
	WasMerged  bool
	MergedFrom *int64
}

func NormalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		v := strings.ToLower(strings.TrimSpace(tag))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func NormalizeCustomFields(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		key := strings.ToLower(strings.TrimSpace(k))
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	return out
}

func NormalizeSkills(skills []string) []string {
	return NormalizeTags(skills)
}

func ParseListCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func shaField(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func nullable(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseFloatOrZero(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	if parsed < 0 {
		return 0
	}
	return parsed
}

func parseKV(input string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(input, ",") {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		if k == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func isInScope(user middleware.AuthUser, institution, department, team string) bool {
	if user.Role == "system_admin" {
		return true
	}
	return user.Institution == institution && user.Department == department && user.Team == team
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func intersectCount(a, b []string) int {
	set := map[string]struct{}{}
	for _, item := range a {
		v := strings.ToLower(strings.TrimSpace(item))
		if v == "" {
			continue
		}
		set[v] = struct{}{}
	}
	count := 0
	for _, item := range b {
		v := strings.ToLower(strings.TrimSpace(item))
		if v == "" {
			continue
		}
		if _, ok := set[v]; ok {
			count++
		}
	}
	return count
}

func educationRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high_school", "high-school", "hs":
		return 1
	case "diploma", "associate", "associate_degree", "associate-degree":
		return 2
	case "bachelor", "bachelors", "bachelor_degree", "bachelor-degree":
		return 3
	case "master", "masters", "master_degree", "master-degree":
		return 4
	case "phd", "doctorate":
		return 5
	default:
		return 0
	}
}

func weightedSkillScore(candidateSkills, required []string) int {
	if len(required) == 0 {
		return 40
	}
	matched := intersectCount(candidateSkills, required)
	ratio := float64(matched) / float64(len(required))
	if ratio > 1 {
		ratio = 1
	}
	return int(ratio * 40)
}

func weightedEducationScore(candidate, required string) int {
	req := strings.ToLower(strings.TrimSpace(required))
	if req == "" {
		return 20
	}
	cand := strings.ToLower(strings.TrimSpace(candidate))
	if cand == req {
		return 20
	}
	candRank := educationRank(cand)
	reqRank := educationRank(req)
	if candRank >= reqRank && candRank > 0 {
		return 17
	}
	if candRank+1 == reqRank {
		return 10
	}
	return 4
}

func weightedExperienceScore(candidate, required float64) int {
	if required <= 0 {
		if candidate <= 0 {
			return 15
		}
		return 25
	}
	ratio := candidate / required
	switch {
	case ratio >= 1.25:
		return 25
	case ratio >= 1:
		return 22
	case ratio >= 0.8:
		return 16
	case ratio >= 0.5:
		return 9
	default:
		return 3
	}
}

func weightedTimeScore(lastActiveAt time.Time, targetDays int) int {
	if targetDays <= 0 {
		targetDays = 30
	}
	days := int(time.Since(lastActiveAt).Hours() / 24)
	if days < 0 {
		days = 0
	}
	switch {
	case days <= targetDays/4:
		return 15
	case days <= targetDays/2:
		return 12
	case days <= targetDays:
		return 9
	case days <= targetDays*2:
		return 5
	default:
		return 2
	}
}

func skillReason(candidateSkills, required []string) string {
	if len(required) == 0 {
		return "position has no mandatory skills"
	}
	matched := intersectCount(candidateSkills, required)
	return fmt.Sprintf("skills matched %d/%d", matched, len(required))
}

func educationReason(candidate, required string) string {
	if strings.TrimSpace(required) == "" {
		return "position has no mandatory education level"
	}
	return fmt.Sprintf("education candidate=%s required=%s", strings.TrimSpace(candidate), strings.TrimSpace(required))
}

func experienceReason(candidate, required float64) string {
	return fmt.Sprintf("experience candidate=%.2f years required=%.2f years", candidate, required)
}

func timeReason(lastActiveAt time.Time, targetDays int) string {
	days := int(time.Since(lastActiveAt).Hours() / 24)
	return fmt.Sprintf("last active %d day(s) ago vs target %d day(s)", days, targetDays)
}

func candidateSimilarity(a, b CandidateModel) (int, []string) {
	score := 0
	reasons := make([]string, 0)

	if strings.EqualFold(a.EducationLevel, b.EducationLevel) && strings.TrimSpace(a.EducationLevel) != "" {
		score += 20
		reasons = append(reasons, "shared education level")
	}
	if absFloat(a.YearsExperience-b.YearsExperience) <= 1.5 {
		score += 20
		reasons = append(reasons, "close experience range")
	}

	sharedSkills := intersectCount(a.Skills, b.Skills)
	if sharedSkills > 0 {
		add := sharedSkills * 12
		if add > 48 {
			add = 48
		}
		score += add
		reasons = append(reasons, fmt.Sprintf("shared skills %d", sharedSkills))
	}

	sharedTags := intersectCount(a.Tags, b.Tags)
	if sharedTags > 0 {
		add := sharedTags * 6
		if add > 12 {
			add = 12
		}
		score += add
		reasons = append(reasons, fmt.Sprintf("shared tags %d", sharedTags))
	}

	if score > 100 {
		score = 100
	}
	return score, reasons
}

func (s *RecruitmentService) findDuplicate(ctx context.Context, user middleware.AuthUser, phoneHash, idHash string) (int64, error) {
	where, args := middleware.BuildScopeWhere(user, "")
	clauses := make([]string, 0, 2)
	queryArgs := make([]any, 0, len(args)+2)

	if phoneHash != "" {
		clauses = append(clauses, "phone_hash = ?")
		queryArgs = append(queryArgs, phoneHash)
	}
	if idHash != "" {
		clauses = append(clauses, "id_number_hash = ?")
		queryArgs = append(queryArgs, idHash)
	}
	if len(clauses) == 0 {
		return 0, sql.ErrNoRows
	}

	query := `SELECT id FROM candidates WHERE (` + strings.Join(clauses, " OR ") + `) AND ` + where + ` ORDER BY id ASC LIMIT 1`
	queryArgs = append(queryArgs, args...)

	var id int64
	if err := s.db.QueryRowContext(ctx, query, queryArgs...).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *RecruitmentService) LoadPosition(ctx context.Context, id int64) (PositionModel, error) {
	var (
		p              PositionModel
		requiredSkills string
		tagsRaw        string
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, COALESCE(description, ''), COALESCE(required_skills_text, ''),
		       COALESCE(required_education_level, ''), COALESCE(min_years_experience, 0),
		       COALESCE(target_time_to_fill_days, 30), COALESCE(CAST(tags_json AS CHAR), '[]'),
		       institution, department, team
		FROM positions
		WHERE id = ?
	`, id).Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&requiredSkills,
		&p.RequiredEducation,
		&p.MinYearsExperience,
		&p.TargetTimeToFillDays,
		&tagsRaw,
		&p.Institution,
		&p.Department,
		&p.Team,
	)
	if err != nil {
		return PositionModel{}, err
	}

	p.RequiredSkills = NormalizeSkills(ParseListCSV(requiredSkills))
	_ = json.Unmarshal([]byte(tagsRaw), &p.Tags)
	return p, nil
}

func (s *RecruitmentService) LoadCandidate(ctx context.Context, id int64) (CandidateModel, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.full_name, c.phone_enc, c.id_number_enc, COALESCE(c.email, ''), COALESCE(c.resume_path, ''),
		       c.position_id, COALESCE(p.title, ''), c.status, COALESCE(CAST(c.tags_json AS CHAR), '[]'),
		       COALESCE(CAST(c.custom_fields_json AS CHAR), '{}'), COALESCE(c.skills_text, ''), COALESCE(c.education_level, ''),
		       COALESCE(c.years_experience, 0), COALESCE(c.last_active_at, UTC_TIMESTAMP()),
		       c.institution, c.department, c.team, c.created_at, c.updated_at
		FROM candidates c
		LEFT JOIN positions p ON p.id = c.position_id
		WHERE c.id = ?
	`, id)

	var (
		model      CandidateModel
		phoneEnc   string
		idEnc      string
		positionID sql.NullInt64
		tagsRaw    string
		customRaw  string
		skillsRaw  string
	)

	if err := row.Scan(
		&model.ID,
		&model.FullName,
		&phoneEnc,
		&idEnc,
		&model.Email,
		&model.ResumePath,
		&positionID,
		&model.PositionTitle,
		&model.Status,
		&tagsRaw,
		&customRaw,
		&skillsRaw,
		&model.EducationLevel,
		&model.YearsExperience,
		&model.LastActiveAt,
		&model.Institution,
		&model.Department,
		&model.Team,
		&model.CreatedAt,
		&model.UpdatedAt,
	); err != nil {
		return CandidateModel{}, err
	}

	phone, err := s.cipher.Decrypt(phoneEnc)
	if err != nil {
		return CandidateModel{}, err
	}
	idNumber, err := s.cipher.Decrypt(idEnc)
	if err != nil {
		return CandidateModel{}, err
	}

	model.Phone = phone
	model.IDNumber = idNumber
	if positionID.Valid {
		pid := positionID.Int64
		model.PositionID = &pid
	}

	_ = json.Unmarshal([]byte(tagsRaw), &model.Tags)
	_ = json.Unmarshal([]byte(customRaw), &model.CustomFields)
	model.Skills = NormalizeSkills(ParseListCSV(skillsRaw))

	return model, nil
}

func (s *RecruitmentService) ListCandidates(ctx context.Context, user middleware.AuthUser) ([]CandidateModel, error) {
	where, args := middleware.BuildScopeWhere(user, "c")
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id
		FROM candidates c
		WHERE `+where+`
		ORDER BY c.id DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CandidateModel, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		item, err := s.LoadCandidate(ctx, id)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *RecruitmentService) UpsertCandidate(ctx context.Context, user middleware.AuthUser, in CandidateUpsertInput) (DuplicateCheckResult, error) {
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)
	idNumber := strings.TrimSpace(in.IDNumber)
	if fullName == "" || phone == "" || idNumber == "" {
		return DuplicateCheckResult{}, fmt.Errorf("full_name, phone and id_number are required")
	}

	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "new"
	}

	tags := NormalizeTags(in.Tags)
	custom := NormalizeCustomFields(in.CustomFields)
	skills := NormalizeSkills(in.Skills)
	education := strings.TrimSpace(in.EducationLevel)
	years := in.YearsExperience
	if years < 0 {
		years = 0
	}

	tagsJSON, _ := json.Marshal(tags)
	customJSON, _ := json.Marshal(custom)
	skillsText := strings.Join(skills, ",")
	phoneHash := shaField(phone)
	idHash := shaField(idNumber)

	phoneEnc, err := s.cipher.Encrypt(phone)
	if err != nil {
		return DuplicateCheckResult{}, fmt.Errorf("encrypt phone: %w", err)
	}
	idEnc, err := s.cipher.Encrypt(idNumber)
	if err != nil {
		return DuplicateCheckResult{}, fmt.Errorf("encrypt id: %w", err)
	}

	lastActive := time.Now().UTC()
	if in.LastActiveAt != nil {
		lastActive = in.LastActiveAt.UTC()
	}

	if in.ID > 0 {
		current, err := s.LoadCandidate(ctx, in.ID)
		if err != nil {
			return DuplicateCheckResult{}, err
		}
		if !isInScope(user, current.Institution, current.Department, current.Team) {
			return DuplicateCheckResult{}, fmt.Errorf("candidate is outside your data scope")
		}

		if _, err := s.db.ExecContext(ctx, `
			UPDATE candidates
			SET full_name = ?, phone_enc = ?, phone_hash = ?, id_number_enc = ?, id_number_hash = ?, email = ?, resume_path = ?,
				position_id = ?, status = ?, tags_json = ?, custom_fields_json = ?, skills_text = ?, education_level = ?,
				years_experience = ?, last_active_at = ?
			WHERE id = ?
		`, fullName, phoneEnc, phoneHash, idEnc, idHash, nullable(in.Email), nullable(in.ResumePath), in.PositionID,
			status, string(tagsJSON), string(customJSON), skillsText, nullable(education), years, lastActive, in.ID); err != nil {
			return DuplicateCheckResult{}, fmt.Errorf("update candidate: %w", err)
		}
		model, err := s.LoadCandidate(ctx, in.ID)
		if err != nil {
			return DuplicateCheckResult{}, err
		}
		return DuplicateCheckResult{Model: model}, nil
	}

	if existingID, err := s.findDuplicate(ctx, user, phoneHash, idHash); err == nil && existingID > 0 {
		in.ID = existingID
		result, err := s.UpsertCandidate(ctx, user, in)
		if err != nil {
			return DuplicateCheckResult{}, err
		}
		result.WasMerged = true
		result.MergedFrom = &existingID
		return result, nil
	}

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO candidates
		(full_name, phone_enc, phone_hash, id_number_enc, id_number_hash, email, resume_path, tags_json, custom_fields_json,
		 skills_text, education_level, years_experience, last_active_at, position_id, institution, department, team, status, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fullName, phoneEnc, phoneHash, idEnc, idHash, nullable(in.Email), nullable(in.ResumePath), string(tagsJSON), string(customJSON),
		skillsText, nullable(education), years, lastActive, in.PositionID,
		user.Institution, user.Department, user.Team, status, user.ID)
	if err != nil {
		return DuplicateCheckResult{}, fmt.Errorf("insert candidate: %w", err)
	}
	id, _ := res.LastInsertId()
	model, err := s.LoadCandidate(ctx, id)
	if err != nil {
		return DuplicateCheckResult{}, err
	}
	return DuplicateCheckResult{Model: model}, nil
}

func (s *RecruitmentService) ImportCandidateRows(ctx context.Context, user middleware.AuthUser, rows []map[string]string) (int, []map[string]any, error) {
	imported := 0
	failed := make([]map[string]any, 0)

	for idx, row := range rows {
		input := CandidateUpsertInput{
			FullName:        strings.TrimSpace(row["full_name"]),
			Phone:           strings.TrimSpace(row["phone"]),
			IDNumber:        strings.TrimSpace(row["id_number"]),
			Email:           strings.TrimSpace(row["email"]),
			Status:          "imported",
			Tags:            ParseListCSV(row["tags"]),
			CustomFields:    parseKV(row["custom_fields"]),
			Skills:          ParseListCSV(row["skills"]),
			EducationLevel:  strings.TrimSpace(row["education_level"]),
			YearsExperience: parseFloatOrZero(row["years_experience"]),
		}
		if input.FullName == "" || input.Phone == "" || input.IDNumber == "" {
			failed = append(failed, map[string]any{"row": idx + 2, "reason": "missing full_name/phone/id_number"})
			continue
		}
		if _, err := s.UpsertCandidate(ctx, user, input); err != nil {
			failed = append(failed, map[string]any{"row": idx + 2, "reason": err.Error()})
			continue
		}
		imported++
	}
	return imported, failed, nil
}

func (s *RecruitmentService) MergeDuplicates(ctx context.Context, user middleware.AuthUser, primaryID int64, duplicateIDs []int64) ([]int64, error) {
	primary, err := s.LoadCandidate(ctx, primaryID)
	if err != nil {
		return nil, err
	}
	if !isInScope(user, primary.Institution, primary.Department, primary.Team) {
		return nil, fmt.Errorf("candidate is outside your data scope")
	}

	merged := make([]int64, 0)
	for _, dupID := range duplicateIDs {
		if dupID == primaryID {
			continue
		}
		dup, err := s.LoadCandidate(ctx, dupID)
		if err != nil {
			continue
		}
		if !isInScope(user, dup.Institution, dup.Department, dup.Team) {
			continue
		}
		if shaField(primary.Phone) != shaField(dup.Phone) && shaField(primary.IDNumber) != shaField(dup.IDNumber) {
			continue
		}
		if err := s.mergeOne(ctx, primaryID, dupID); err != nil {
			continue
		}
		merged = append(merged, dupID)
	}
	return merged, nil
}

func (s *RecruitmentService) mergeOne(ctx context.Context, primaryID, duplicateID int64) error {
	primary, err := s.LoadCandidate(ctx, primaryID)
	if err != nil {
		return err
	}
	dup, err := s.LoadCandidate(ctx, duplicateID)
	if err != nil {
		return err
	}

	merged := primary
	if strings.TrimSpace(merged.Email) == "" {
		merged.Email = dup.Email
	}
	if strings.TrimSpace(merged.ResumePath) == "" {
		merged.ResumePath = dup.ResumePath
	}
	if merged.PositionID == nil {
		merged.PositionID = dup.PositionID
	}
	merged.Tags = NormalizeTags(append(merged.Tags, dup.Tags...))
	if merged.CustomFields == nil {
		merged.CustomFields = map[string]string{}
	}
	for k, v := range dup.CustomFields {
		if strings.TrimSpace(merged.CustomFields[k]) == "" {
			merged.CustomFields[k] = v
		}
	}
	merged.Skills = NormalizeSkills(append(merged.Skills, dup.Skills...))
	if strings.TrimSpace(merged.EducationLevel) == "" {
		merged.EducationLevel = dup.EducationLevel
	}
	if merged.YearsExperience <= 0 {
		merged.YearsExperience = dup.YearsExperience
	}
	if dup.LastActiveAt.After(merged.LastActiveAt) {
		merged.LastActiveAt = dup.LastActiveAt
	}

	if _, err := s.UpsertCandidate(ctx, middleware.AuthUser{
		ID:          0,
		Role:        "system_admin",
		Institution: merged.Institution,
		Department:  merged.Department,
		Team:        merged.Team,
	}, CandidateUpsertInput{
		ID:              merged.ID,
		FullName:        merged.FullName,
		Phone:           merged.Phone,
		IDNumber:        merged.IDNumber,
		Email:           merged.Email,
		ResumePath:      merged.ResumePath,
		PositionID:      merged.PositionID,
		Status:          merged.Status,
		Tags:            merged.Tags,
		CustomFields:    merged.CustomFields,
		Skills:          merged.Skills,
		EducationLevel:  merged.EducationLevel,
		YearsExperience: merged.YearsExperience,
		LastActiveAt:    &merged.LastActiveAt,
	}); err != nil {
		return err
	}

	_, _ = s.db.ExecContext(ctx, `UPDATE attachments SET record_id = ? WHERE module_name = 'candidates' AND record_id = ?`, primaryID, duplicateID)
	_, err = s.db.ExecContext(ctx, `DELETE FROM candidates WHERE id = ?`, duplicateID)
	return err
}

func ScoreCandidate(tokens []string, candidate CandidateModel) (int, []string) {
	score := 0
	reasons := make([]string, 0)

	nameLower := strings.ToLower(candidate.FullName)
	emailLower := strings.ToLower(candidate.Email)
	phoneLower := strings.ToLower(candidate.Phone)
	idLower := strings.ToLower(candidate.IDNumber)
	skillsLower := strings.ToLower(strings.Join(candidate.Skills, " "))
	eduLower := strings.ToLower(candidate.EducationLevel)
	expText := fmt.Sprintf("%.2f", candidate.YearsExperience)
	daysSinceActive := int(time.Since(candidate.LastActiveAt).Hours() / 24)
	if daysSinceActive < 0 {
		daysSinceActive = 0
	}

	for _, token := range tokens {
		if strings.Contains(nameLower, token) {
			score += 24
			reasons = append(reasons, fmt.Sprintf("name matched '%s' (+24)", token))
		}
		if emailLower != "" && strings.Contains(emailLower, token) {
			score += 8
			reasons = append(reasons, fmt.Sprintf("email matched '%s' (+8)", token))
		}
		if strings.Contains(phoneLower, token) {
			score += 14
			reasons = append(reasons, fmt.Sprintf("phone matched '%s' (+14)", token))
		}
		if strings.Contains(idLower, token) {
			score += 14
			reasons = append(reasons, fmt.Sprintf("id_number matched '%s' (+14)", token))
		}
		if skillsLower != "" && strings.Contains(skillsLower, token) {
			score += 20
			reasons = append(reasons, fmt.Sprintf("skills matched '%s' (+20)", token))
		}
		if eduLower != "" && strings.Contains(eduLower, token) {
			score += 10
			reasons = append(reasons, fmt.Sprintf("education matched '%s' (+10)", token))
		}
		if strings.Contains(expText, token) {
			score += 10
			reasons = append(reasons, fmt.Sprintf("experience matched '%s' (+10)", token))
		}
	}

	if score > 0 {
		timeBonus := 0
		switch {
		case daysSinceActive <= 7:
			timeBonus = 10
		case daysSinceActive <= 30:
			timeBonus = 6
		case daysSinceActive <= 90:
			timeBonus = 3
		default:
			timeBonus = 1
		}
		score += timeBonus
		reasons = append(reasons, fmt.Sprintf("recent activity contributed (+%d)", timeBonus))
	}

	if score > 100 {
		score = 100
	}
	return score, reasons
}

func (s *RecruitmentService) SmartSearch(ctx context.Context, user middleware.AuthUser, query string) ([]SearchCandidate, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return []SearchCandidate{}, nil
	}
	tokens := strings.Fields(query)
	candidates, err := s.ListCandidates(ctx, user)
	if err != nil {
		return nil, err
	}

	results := make([]SearchCandidate, 0)
	for _, candidate := range candidates {
		score, reasons := ScoreCandidate(tokens, candidate)
		if score == 0 {
			continue
		}
		results = append(results, SearchCandidate{
			CandidateID: candidate.ID,
			FullName:    candidate.FullName,
			MaskedPhone: security.MaskPhone(candidate.Phone),
			MaskedID:    security.MaskID(candidate.IDNumber),
			Score:       score,
			Reasons:     reasons,
			Institution: candidate.Institution,
			Department:  candidate.Department,
			Team:        candidate.Team,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].CandidateID > results[j].CandidateID
		}
		return results[i].Score > results[j].Score
	})
	return results, nil
}

func (s *RecruitmentService) ExplainableMatch(ctx context.Context, user middleware.AuthUser, candidateID, positionID int64) (MatchScoreBreakdown, error) {
	candidate, err := s.LoadCandidate(ctx, candidateID)
	if err != nil {
		return MatchScoreBreakdown{}, err
	}
	position, err := s.LoadPosition(ctx, positionID)
	if err != nil {
		return MatchScoreBreakdown{}, err
	}

	if !isInScope(user, candidate.Institution, candidate.Department, candidate.Team) {
		return MatchScoreBreakdown{}, fmt.Errorf("candidate is outside your data scope")
	}
	if !isInScope(user, position.Institution, position.Department, position.Team) {
		return MatchScoreBreakdown{}, fmt.Errorf("position is outside your data scope")
	}

	skillScore := weightedSkillScore(candidate.Skills, position.RequiredSkills)
	educationScore := weightedEducationScore(candidate.EducationLevel, position.RequiredEducation)
	experienceScore := weightedExperienceScore(candidate.YearsExperience, position.MinYearsExperience)
	timeScore := weightedTimeScore(candidate.LastActiveAt, position.TargetTimeToFillDays)

	total := skillScore + educationScore + experienceScore + timeScore
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}

	return MatchScoreBreakdown{
		Score: total,
		Weighted: []string{
			fmt.Sprintf("skills 40%% => %d/40", skillScore),
			fmt.Sprintf("education 20%% => %d/20", educationScore),
			fmt.Sprintf("experience 25%% => %d/25", experienceScore),
			fmt.Sprintf("time 15%% => %d/15", timeScore),
		},
		Reasons: []string{
			skillReason(candidate.Skills, position.RequiredSkills),
			educationReason(candidate.EducationLevel, position.RequiredEducation),
			experienceReason(candidate.YearsExperience, position.MinYearsExperience),
			timeReason(candidate.LastActiveAt, position.TargetTimeToFillDays),
		},
		SkillScore:      skillScore,
		EducationScore:  educationScore,
		ExperienceScore: experienceScore,
		TimeScore:       timeScore,
	}, nil
}

func (s *RecruitmentService) SimilarCandidates(ctx context.Context, user middleware.AuthUser, candidateID int64, limit int) ([]SimilarCandidate, error) {
	base, err := s.LoadCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	if !isInScope(user, base.Institution, base.Department, base.Team) {
		return nil, fmt.Errorf("candidate is outside your data scope")
	}
	candidates, err := s.ListCandidates(ctx, user)
	if err != nil {
		return nil, err
	}

	out := make([]SimilarCandidate, 0)
	for _, cand := range candidates {
		if cand.ID == candidateID {
			continue
		}
		similarity, reasons := candidateSimilarity(base, cand)
		if similarity <= 0 {
			continue
		}
		out = append(out, SimilarCandidate{
			CandidateID:   cand.ID,
			FullName:      cand.FullName,
			PositionTitle: cand.PositionTitle,
			Similarity:    similarity,
			Reasons:       reasons,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Similarity == out[j].Similarity {
			return out[i].CandidateID > out[j].CandidateID
		}
		return out[i].Similarity > out[j].Similarity
	})

	if limit <= 0 || limit > 20 {
		limit = 5
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *RecruitmentService) SimilarPositions(ctx context.Context, user middleware.AuthUser, candidateID int64, limit int) ([]SimilarPosition, error) {
	candidate, err := s.LoadCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	if !isInScope(user, candidate.Institution, candidate.Department, candidate.Team) {
		return nil, fmt.Errorf("candidate is outside your data scope")
	}

	where, args := middleware.BuildScopeWhere(user, "")
	rows, err := s.db.QueryContext(ctx, `
		SELECT id
		FROM positions
		WHERE `+where+`
		ORDER BY id DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SimilarPosition, 0)
	for rows.Next() {
		var positionID int64
		if err := rows.Scan(&positionID); err != nil {
			continue
		}
		position, err := s.LoadPosition(ctx, positionID)
		if err != nil {
			continue
		}
		match, err := s.ExplainableMatch(ctx, user, candidate.ID, position.ID)
		if err != nil {
			continue
		}
		if match.Score == 0 {
			continue
		}
		out = append(out, SimilarPosition{
			PositionID: position.ID,
			Title:      position.Title,
			Similarity: match.Score,
			Reasons:    match.Reasons,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Similarity == out[j].Similarity {
			return out[i].PositionID > out[j].PositionID
		}
		return out[i].Similarity > out[j].Similarity
	})

	if limit <= 0 || limit > 20 {
		limit = 5
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
