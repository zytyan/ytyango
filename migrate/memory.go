package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SampleConfig struct {
	Rate float64
	Rows int
}

func copyToMemory(ctx context.Context, sourcePath string, sample SampleConfig, logf func(string, ...any)) (*sql.DB, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path is empty")
	}
	src, err := openSQLite(ctx, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open source db: %w", err)
	}
	defer src.Close()

	memDSN := fmt.Sprintf("file:migrate_mem_%d?mode=memory&cache=shared", time.Now().UnixNano())
	dst, err := openSQLite(ctx, memDSN)
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	if _, err := dst.ExecContext(ctx, "PRAGMA foreign_keys=OFF;"); err != nil {
		_ = dst.Close()
		return nil, fmt.Errorf("disable foreign keys: %w", err)
	}

	createOrder := []string{"table", "view", "index", "trigger"}
	for _, typ := range createOrder {
		objs, err := loadSchemaObjects(ctx, src, typ)
		if err != nil {
			_ = dst.Close()
			return nil, err
		}
		for _, obj := range objs {
			if obj.SQL == "" {
				continue
			}
			if _, err := dst.ExecContext(ctx, obj.SQL); err != nil {
				_ = dst.Close()
				return nil, fmt.Errorf("create %s %s: %w", typ, obj.Name, err)
			}
		}
	}

	if err := attachAndCopyData(ctx, dst, sourcePath, sample, logf); err != nil {
		_ = dst.Close()
		return nil, err
	}

	if _, err := dst.ExecContext(ctx, "PRAGMA foreign_keys=ON;"); err != nil && logf != nil {
		logf("re-enable foreign keys in memory db: %v\n", err)
	}
	return dst, nil
}

type schemaObject struct {
	Name string
	SQL  string
}

func loadSchemaObjects(ctx context.Context, db *sql.DB, typ string) ([]schemaObject, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, sql FROM sqlite_master WHERE type = ? AND name NOT LIKE 'sqlite_%' ORDER BY name;`, typ)
	if err != nil {
		return nil, fmt.Errorf("load schema objects: %w", err)
	}
	defer rows.Close()
	var objs []schemaObject
	for rows.Next() {
		var obj schemaObject
		if err := rows.Scan(&obj.Name, &obj.SQL); err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return objs, nil
}

func attachAndCopyData(ctx context.Context, dst *sql.DB, sourcePath string, sample SampleConfig, logf func(string, ...any)) error {
	if _, err := dst.ExecContext(ctx, "ATTACH DATABASE ? AS diskdb;", sourcePath); err != nil {
		return fmt.Errorf("attach source database: %w", err)
	}
	defer dst.ExecContext(ctx, "DETACH DATABASE diskdb;")

	tables, err := loadSchemaObjects(ctx, dst, "table")
	if err != nil {
		return err
	}
	sampleClause := buildSampleClause(sample)
	for _, tbl := range tables {
		// Skip internal tables that sneak past filter.
		if strings.HasPrefix(tbl.Name, "sqlite_") {
			continue
		}
		sqlStmt := fmt.Sprintf("INSERT INTO %s SELECT * FROM diskdb.%s%s;", quoteIdent(tbl.Name), quoteIdent(tbl.Name), sampleClause)
		if logf != nil {
			logf("[memory-run] copy table %s %s\n", tbl.Name, strings.TrimSpace(sampleClause))
		}
		if _, err := dst.ExecContext(ctx, sqlStmt); err != nil {
			return fmt.Errorf("copy table %s: %w", tbl.Name, err)
		}
	}
	return nil
}

func buildSampleClause(sample SampleConfig) string {
	if sample.Rows > 0 {
		return fmt.Sprintf(" LIMIT %d", sample.Rows)
	}
	if sample.Rate > 0 && sample.Rate < 1 {
		threshold := int(sample.Rate * 1000000)
		if threshold <= 0 {
			threshold = 1
		}
		return fmt.Sprintf(" WHERE (abs(random()) %% 1000000) < %d", threshold)
	}
	return ""
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
