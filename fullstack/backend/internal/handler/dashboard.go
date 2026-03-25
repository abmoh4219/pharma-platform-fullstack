package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pharma-platform/internal/middleware"
)

func (a *API) DashboardSummary(c *gin.Context) {
	user, _ := middleware.GetAuthUser(c)
	candidateCount := a.countScoped("candidates", user)
	openCases := a.countScopedWithCondition("case_ledgers", user, "status IN ('new','assigned','in_progress')")
	expiringQualifications := a.countScopedWithCondition("qualifications", user, "expiry_date <= DATE_ADD(CURDATE(), INTERVAL 30 DAY)")
	activeRestrictions := a.countScopedWithCondition("restrictions", user, "is_active = 1")

	writeSuccess(c, http.StatusOK, gin.H{
		"role":                    user.Role,
		"scope":                   gin.H{"institution": user.Institution, "department": user.Department, "team": user.Team},
		"candidates":              candidateCount,
		"open_cases":              openCases,
		"expiring_qualifications": expiringQualifications,
		"active_restrictions":     activeRestrictions,
	})
}

func (a *API) countScoped(table string, user middleware.AuthUser) int {
	return a.countScopedWithCondition(table, user, "1=1")
}

func (a *API) countScopedWithCondition(table string, user middleware.AuthUser, condition string) int {
	if user.Role == "system_admin" {
		var count int
		_ = a.db.QueryRow("SELECT COUNT(1) FROM " + table + " WHERE " + condition).Scan(&count)
		return count
	}
	var count int
	_ = a.db.QueryRow(
		"SELECT COUNT(1) FROM "+table+" WHERE institution = ? AND department = ? AND team = ? AND "+condition,
		user.Institution,
		user.Department,
		user.Team,
	).Scan(&count)
	return count
}
