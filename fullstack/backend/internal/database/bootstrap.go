package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func ApplyInitSQL(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read init sql: %w", err)
	}

	cleaned := stripSQLComments(string(content))
	statements := strings.Split(cleaned, ";")
	for _, stmt := range statements {
		sqlStmt := strings.TrimSpace(stmt)
		if sqlStmt == "" {
			continue
		}
		if _, err := db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("exec init statement: %w; statement=%s", err, compact(sqlStmt, 120))
		}
	}

	return nil
}

func stripSQLComments(input string) string {
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func compact(value string, max int) string {
	s := strings.Join(strings.Fields(value), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
