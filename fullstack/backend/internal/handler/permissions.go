package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
)

type updateUserPermissionRequest struct {
	RoleCode    string `json:"role_code" binding:"required,min=1,max=64"`
	Institution string `json:"institution" binding:"required,min=1,max=128"`
	Department  string `json:"department" binding:"required,min=1,max=128"`
	Team        string `json:"team" binding:"required,min=1,max=128"`
	Reason      string `json:"reason" binding:"omitempty,max=255"`
}

func (a *API) UpdateUserPermission(c *gin.Context) {
	actor, _ := middleware.GetAuthUser(c)
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetID <= 0 {
		badRequest(c, "INVALID_ID", "invalid user id")
		return
	}

	var req updateUserPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "INVALID_PAYLOAD", "invalid permission payload")
		return
	}

	req.RoleCode = strings.TrimSpace(req.RoleCode)
	req.Institution = strings.TrimSpace(req.Institution)
	req.Department = strings.TrimSpace(req.Department)
	req.Team = strings.TrimSpace(req.Team)
	req.Reason = strings.TrimSpace(req.Reason)

	before, err := a.loadUserPermissionSnapshot(c, targetID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load user permissions")
		return
	}

	var roleID int64
	if err := a.db.QueryRow(`SELECT id FROM roles WHERE code = ?`, req.RoleCode).Scan(&roleID); err != nil {
		if err == sql.ErrNoRows {
			badRequest(c, "INVALID_ROLE", "unknown role code")
			return
		}
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to resolve role")
		return
	}

	scopeID, err := a.resolveOrCreateScope(req.Institution, req.Department, req.Team)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to resolve data scope")
		return
	}

	if _, err := a.db.Exec(`UPDATE users SET role_id = ?, data_scope_id = ? WHERE id = ?`, roleID, scopeID, targetID); err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update user permissions")
		return
	}

	after, err := a.loadUserPermissionSnapshot(c, targetID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load updated user permissions")
		return
	}

	a.logPermissionChange(c, actor.ID, "users", strconv.FormatInt(targetID, 10), before, after, gin.H{
		"reason":     req.Reason,
		"changed_by": actor.Username,
	})

	writeSuccess(c, http.StatusOK, gin.H{
		"id":          targetID,
		"permissions": after,
	})
}

func (a *API) resolveOrCreateScope(institution, department, team string) (int64, error) {
	var scopeID int64
	err := a.db.QueryRow(`
		SELECT id
		FROM data_scopes
		WHERE institution = ? AND department = ? AND team = ?
	`, institution, department, team).Scan(&scopeID)
	if err == nil {
		return scopeID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	res, err := a.db.Exec(`
		INSERT INTO data_scopes (institution, department, team)
		VALUES (?, ?, ?)
	`, institution, department, team)
	if err != nil {
		if err := a.db.QueryRow(`
			SELECT id
			FROM data_scopes
			WHERE institution = ? AND department = ? AND team = ?
		`, institution, department, team).Scan(&scopeID); err == nil {
			return scopeID, nil
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (a *API) loadUserPermissionSnapshot(c *gin.Context, userID int64) (gin.H, error) {
	var (
		id          int64
		username    string
		fullName    string
		roleCode    string
		institution string
		department  string
		team        string
	)

	err := a.db.QueryRow(`
		SELECT u.id, u.username, u.full_name, r.code, ds.institution, ds.department, ds.team
		FROM users u
		JOIN roles r ON r.id = u.role_id
		JOIN data_scopes ds ON ds.id = u.data_scope_id
		WHERE u.id = ?
	`, userID).Scan(&id, &username, &fullName, &roleCode, &institution, &department, &team)
	if err != nil {
		return nil, err
	}

	return gin.H{
		"id":          id,
		"username":    username,
		"full_name":   fullName,
		"role_code":   roleCode,
		"institution": institution,
		"department":  department,
		"team":        team,
	}, nil
}
