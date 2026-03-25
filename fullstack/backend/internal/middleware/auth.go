package middleware

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/config"
	"pharma-platform/internal/security"
)

const ContextAuthUser = "auth_user"
const ContextClaims = "auth_claims"

type AuthUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
	ScopeID     int64  `json:"scope_id"`
	Institution string `json:"institution"`
	Department  string `json:"department"`
	Team        string `json:"team"`
}

type AuthMiddleware struct {
	db  *sql.DB
	cfg config.Config
}

func NewAuthMiddleware(db *sql.DB, cfg config.Config) *AuthMiddleware {
	return &AuthMiddleware{db: db, cfg: cfg}
}

func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}
		token := strings.TrimSpace(parts[1])
		if token == "" {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}

		claims, err := security.ParseToken(m.cfg.JWTSecret, token)
		if err != nil {
			if errors.Is(err, security.ErrExpiredToken) {
				AbortWithError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "token has expired")
				return
			}
			AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
			return
		}

		var blacklisted int
		if err := m.db.QueryRow(`SELECT COUNT(1) FROM token_blacklist WHERE jti = ? AND expires_at >= UTC_TIMESTAMP()`, claims.ID).Scan(&blacklisted); err != nil {
			AbortWithError(c, http.StatusInternalServerError, "DB_ERROR", "failed to validate token")
			return
		}
		if blacklisted > 0 {
			AbortWithError(c, http.StatusUnauthorized, "TOKEN_REVOKED", "token has been revoked")
			return
		}

		uid, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			AbortWithError(c, http.StatusUnauthorized, "INVALID_TOKEN_SUBJECT", "invalid token subject")
			return
		}

		const query = `
			SELECT u.id, u.username, u.full_name, r.code, ds.id, ds.institution, ds.department, ds.team
			FROM users u
			JOIN roles r ON r.id = u.role_id
			JOIN data_scopes ds ON ds.id = u.data_scope_id
			WHERE u.id = ? AND u.is_active = 1`

		var user AuthUser
		if err := m.db.QueryRow(query, uid).Scan(
			&user.ID,
			&user.Username,
			&user.FullName,
			&user.Role,
			&user.ScopeID,
			&user.Institution,
			&user.Department,
			&user.Team,
		); err != nil {
			if err == sql.ErrNoRows {
				AbortWithError(c, http.StatusUnauthorized, "USER_NOT_FOUND", "user is inactive or missing")
				return
			}
			AbortWithError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load user context")
			return
		}

		c.Set(ContextAuthUser, user)
		c.Set(ContextClaims, claims)
		c.Next()
	}
}

func DataScopeRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := GetAuthUser(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth user context")
			return
		}

		if user.Role != "system_admin" && (user.Institution == "" || user.Department == "" || user.Team == "") {
			AbortWithError(c, http.StatusForbidden, "INVALID_SCOPE", "user data scope is incomplete")
			return
		}

		c.Next()
	}
}

func RequireRoles(allowed ...string) gin.HandlerFunc {
	set := map[string]struct{}{}
	for _, role := range allowed {
		set[role] = struct{}{}
	}

	return func(c *gin.Context) {
		user, ok := GetAuthUser(c)
		if !ok {
			AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth user context")
			return
		}
		if _, ok := set[user.Role]; !ok {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "insufficient role permission")
			return
		}
		c.Next()
	}
}

func GetAuthUser(c *gin.Context) (AuthUser, bool) {
	value, ok := c.Get(ContextAuthUser)
	if !ok {
		return AuthUser{}, false
	}
	user, ok := value.(AuthUser)
	return user, ok
}

func GetClaims(c *gin.Context) (*security.Claims, bool) {
	value, ok := c.Get(ContextClaims)
	if !ok {
		return nil, false
	}
	claims, ok := value.(*security.Claims)
	return claims, ok
}

func BuildScopeWhere(user AuthUser, alias string) (string, []any) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	if user.Role == "system_admin" {
		return "1=1", nil
	}
	return prefix + "institution = ? AND " + prefix + "department = ? AND " + prefix + "team = ?",
		[]any{user.Institution, user.Department, user.Team}
}
