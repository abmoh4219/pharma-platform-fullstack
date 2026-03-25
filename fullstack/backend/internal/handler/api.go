package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"pharma-platform/internal/config"
	"pharma-platform/internal/middleware"
	"pharma-platform/internal/security"
)

type API struct {
	db     *sql.DB
	cfg    config.Config
	cipher *security.FieldCipher
}

func NewAPI(cfg config.Config, db *sql.DB) (*API, error) {
	cipher, err := security.NewFieldCipher(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("init field cipher: %w", err)
	}

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	if err := os.MkdirAll(cfg.UploadTmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload tmp dir: %w", err)
	}

	return &API{db: db, cfg: cfg, cipher: cipher}, nil
}

func (a *API) Health(c *gin.Context) {
	dbStatus := "up"
	status := "ok"
	if err := a.db.Ping(); err != nil {
		dbStatus = "down"
		status = "degraded"
	}
	writeSuccess(c, http.StatusOK, gin.H{
		"status":   status,
		"database": dbStatus,
		"service":  "go-backend",
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *API) Me(c *gin.Context) {
	user, ok := middleware.GetAuthUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "user context not found")
		return
	}
	writeSuccess(c, http.StatusOK, user)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *API) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid login payload")
		return
	}
	if len(req.Password) < 8 {
		badRequest(c, "PASSWORD_TOO_SHORT", "password must be at least 8 characters")
		return
	}

	const query = `
		SELECT u.id, u.username, u.full_name, u.password_hash, r.code, ds.id, ds.institution, ds.department, ds.team
		FROM users u
		JOIN roles r ON r.id = u.role_id
		JOIN data_scopes ds ON ds.id = u.data_scope_id
		WHERE u.username = ? AND u.is_active = 1`

	var user middleware.AuthUser
	var passwordHash string
	if err := a.db.QueryRow(query, req.Username).Scan(
		&user.ID,
		&user.Username,
		&user.FullName,
		&passwordHash,
		&user.Role,
		&user.ScopeID,
		&user.Institution,
		&user.Department,
		&user.Team,
	); err != nil {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		return
	}

	token, jti, expiresAt, err := security.IssueToken(a.cfg.JWTSecret, security.TokenInput{
		UserID:      user.ID,
		Username:    user.Username,
		Role:        user.Role,
		ScopeID:     user.ScopeID,
		Institution: user.Institution,
		Department:  user.Department,
		Team:        user.Team,
		ExpiryHours: a.cfg.JWTExpHours,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "failed to issue access token")
		return
	}

	a.logAudit(c, user.ID, "auth.login", "auth", strconv.FormatInt(user.ID, 10), gin.H{
		"username": user.Username,
		"jti":      jti,
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expiresAt.Format(time.RFC3339),
		"user":         user,
	})
}

func (a *API) Logout(c *gin.Context) {
	user, ok := middleware.GetAuthUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "user context not found")
		return
	}
	claims, ok := middleware.GetClaims(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "token claims not found")
		return
	}

	expiresAt := time.Now().UTC()
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	_, err := a.db.Exec(`
		INSERT INTO token_blacklist (jti, user_id, expires_at)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE expires_at = VALUES(expires_at)
	`, claims.ID, user.ID, expiresAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "LOGOUT_FAILED", "failed to invalidate token")
		return
	}

	a.logAudit(c, user.ID, "auth.logout", "auth", strconv.FormatInt(user.ID, 10), gin.H{
		"username": user.Username,
		"jti":      claims.ID,
	})

	writeSuccess(c, http.StatusOK, gin.H{"message": "logged out"})
}

func (a *API) logAudit(c *gin.Context, userID int64, action, moduleName, recordID string, details any) {
	detailsBytes, _ := json.Marshal(details)
	_, _ = a.db.Exec(`
		INSERT INTO audit_logs (user_id, action, module_name, record_id, details_json, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, action, moduleName, recordID, string(detailsBytes), c.ClientIP(), c.GetHeader("User-Agent"))
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func NormalizeInstitutionPart(value string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9]+`)
	parts := re.ReplaceAllString(strings.ToUpper(strings.TrimSpace(value)), "")
	if parts == "" {
		return "INST"
	}
	if len(parts) > 12 {
		return parts[:12]
	}
	return parts
}

func strPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}
