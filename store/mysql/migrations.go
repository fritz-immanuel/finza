package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func RunMigrations(ctx context.Context, db *sql.DB, schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema file: %w", err)
	}

	stmts := strings.Split(string(data), ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("run migration statement: %w", err)
		}
	}

	return nil
}
