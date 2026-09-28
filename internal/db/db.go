// Package db открывает соединение с SQLite и накатывает схему.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Open открывает (создавая при необходимости) файл базы данных по пути path
// и применяет схему.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite поддерживает только одного писателя одновременно.
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return conn, nil
}

// migrate накатывает точечные изменения схемы, которые CREATE TABLE IF NOT
// EXISTS не подхватывает на уже существующей базе. Каждый шаг безопасно
// пропускается, если уже применён.
func migrate(conn *sql.DB) error {
	steps := []string{
		`ALTER TABLE tool_variants DROP COLUMN min_quantity`,
	}
	for _, stmt := range steps {
		if _, err := conn.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "no such column") {
				continue
			}
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}
