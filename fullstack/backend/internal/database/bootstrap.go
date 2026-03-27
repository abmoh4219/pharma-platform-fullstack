package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

var addColumnIfNotExistsRE = regexp.MustCompile(`(?i)\bADD\s+COLUMN\s+IF\s+NOT\s+EXISTS\s+`)

func ApplyInitSQL(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read init sql: %w", err)
	}

	statements := splitSQLStatements(string(content))
	for _, stmt := range statements {
		sqlStmt := strings.TrimSpace(stmt)
		if sqlStmt == "" {
			continue
		}

		normalizedStmt, hadIfNotExists := normalizeAddColumnSyntax(sqlStmt)
		if _, err := db.Exec(normalizedStmt); err != nil {
			if hadIfNotExists && isMySQLDuplicateColumn(err) {
				continue
			}
			return fmt.Errorf("exec init statement: %w; statement=%s", err, compact(normalizedStmt, 120))
		}
	}

	return nil
}

func normalizeAddColumnSyntax(statement string) (string, bool) {
	normalized := addColumnIfNotExistsRE.ReplaceAllString(statement, "ADD COLUMN ")
	return normalized, normalized != statement
}

func isMySQLDuplicateColumn(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1060
	}
	return false
}

func splitSQLStatements(input string) []string {
	delimiter := ";"
	lines := strings.Split(input, "\n")
	statements := make([]string, 0)
	current := strings.Builder{}

	inSingle := false
	inDouble := false
	inBacktick := false
	inBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inSingle && !inDouble && !inBacktick && !inBlockComment {
			upper := strings.ToUpper(trimmed)
			if strings.HasPrefix(upper, "DELIMITER ") {
				next := strings.TrimSpace(trimmed[len("DELIMITER "):])
				if next == "" {
					next = ";"
				}
				delimiter = next
				continue
			}
			if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
				continue
			}
		}

		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)

		if lineEndsWithDelimiter(line, delimiter, &inSingle, &inDouble, &inBacktick, &inBlockComment) {
			stmt := trimTrailingDelimiter(current.String(), delimiter)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	if trailing := strings.TrimSpace(current.String()); trailing != "" {
		statements = append(statements, trailing)
	}

	return statements
}

func lineEndsWithDelimiter(line, delimiter string, inSingle, inDouble, inBacktick, inBlockComment *bool) bool {
	if delimiter == "" {
		delimiter = ";"
	}

	lastDelimiter := -1
	for i := 0; i < len(line); i++ {
		c := line[i]

		if *inBlockComment {
			if i+1 < len(line) && c == '*' && line[i+1] == '/' {
				*inBlockComment = false
				i++
			}
			continue
		}

		if *inSingle {
			if c == '\\' {
				i++
				continue
			}
			if c == '\'' {
				*inSingle = false
			}
			continue
		}

		if *inDouble {
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				*inDouble = false
			}
			continue
		}

		if *inBacktick {
			if c == '`' {
				*inBacktick = false
			}
			continue
		}

		if i+1 < len(line) && c == '/' && line[i+1] == '*' {
			*inBlockComment = true
			i++
			continue
		}

		if c == '#' {
			break
		}
		if i+1 < len(line) && c == '-' && line[i+1] == '-' {
			if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
				break
			}
		}

		switch c {
		case '\'':
			*inSingle = true
			continue
		case '"':
			*inDouble = true
			continue
		case '`':
			*inBacktick = true
			continue
		}

		if i+len(delimiter) <= len(line) && line[i:i+len(delimiter)] == delimiter {
			lastDelimiter = i
			i += len(delimiter) - 1
		}
	}

	if *inSingle || *inDouble || *inBacktick || *inBlockComment {
		return false
	}
	if lastDelimiter < 0 {
		return false
	}

	tail := strings.TrimSpace(line[lastDelimiter+len(delimiter):])
	return tail == "" || strings.HasPrefix(tail, "--") || strings.HasPrefix(tail, "#")
}

func trimTrailingDelimiter(statement, delimiter string) string {
	trimmed := strings.TrimSpace(statement)
	if delimiter == "" {
		delimiter = ";"
	}
	if strings.HasSuffix(trimmed, delimiter) {
		trimmed = strings.TrimSpace(trimmed[:len(trimmed)-len(delimiter)])
	}
	return trimmed
}

func compact(value string, max int) string {
	s := strings.Join(strings.Fields(value), " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
