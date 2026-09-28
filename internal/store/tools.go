package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/romus204/tooler/internal/models"
)

type ToolFilter struct {
	CategoryID *int64
	Search     string
}

const toolListQuery = `
SELECT
	t.id, t.category_id, t.name, t.brand, t.description, t.unit, t.created_at, t.updated_at,
	COALESCE(c.name, ''),
	COUNT(v.id),
	COALESCE(SUM(v.quantity), 0)
FROM tools t
LEFT JOIN categories c ON c.id = t.category_id
LEFT JOIN tool_variants v ON v.tool_id = t.id
`

func (s *Store) ListTools(ctx context.Context, filter ToolFilter) ([]models.ToolWithStock, error) {
	query := toolListQuery
	var args []any
	var conds []string

	if filter.CategoryID != nil {
		conds = append(conds, "t.category_id = ?")
		args = append(args, *filter.CategoryID)
	}
	if filter.Search != "" {
		conds = append(conds, "(t.name LIKE ? OR t.brand LIKE ? OR t.description LIKE ?)")
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like)
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " GROUP BY t.id ORDER BY t.name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	defer rows.Close()

	var out []models.ToolWithStock
	for rows.Next() {
		var tws models.ToolWithStock
		var categoryID sql.NullInt64
		if err := rows.Scan(
			&tws.ID, &categoryID, &tws.Name, &tws.Brand, &tws.Description, &tws.Unit,
			&tws.CreatedAt, &tws.UpdatedAt, &tws.CategoryName, &tws.VariantCount,
			&tws.TotalQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan tool: %w", err)
		}
		tws.CategoryID = ptrFromNull(categoryID)
		out = append(out, tws)
	}
	return out, rows.Err()
}

func (s *Store) GetTool(ctx context.Context, id int64) (models.Tool, error) {
	var t models.Tool
	var categoryID sql.NullInt64
	row := s.db.QueryRowContext(ctx, `SELECT id, category_id, name, brand, description, unit, created_at, updated_at FROM tools WHERE id = ?`, id)
	if err := row.Scan(&t.ID, &categoryID, &t.Name, &t.Brand, &t.Description, &t.Unit, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return t, mapNotFound(err)
	}
	t.CategoryID = ptrFromNull(categoryID)
	return t, nil
}

func (s *Store) GetToolDetail(ctx context.Context, id int64) (models.ToolDetail, error) {
	var d models.ToolDetail
	tool, err := s.GetTool(ctx, id)
	if err != nil {
		return d, err
	}
	d.Tool = tool

	if tool.CategoryID != nil {
		cat, err := s.GetCategory(ctx, *tool.CategoryID)
		if err == nil {
			d.CategoryName = cat.Name
		}
	}

	variants, err := s.ListVariantsForTool(ctx, id)
	if err != nil {
		return d, err
	}
	d.Variants = variants
	return d, nil
}

func (s *Store) CreateTool(ctx context.Context, t models.Tool) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tools (category_id, name, brand, description, unit) VALUES (?, ?, ?, ?, ?)`,
		nullInt64(t.CategoryID), t.Name, t.Brand, t.Description, t.Unit,
	)
	if err != nil {
		return 0, fmt.Errorf("create tool: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateTool(ctx context.Context, t models.Tool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tools SET category_id = ?, name = ?, brand = ?, description = ?, unit = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		nullInt64(t.CategoryID), t.Name, t.Brand, t.Description, t.Unit, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update tool: %w", err)
	}
	return nil
}

func (s *Store) DeleteTool(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tools WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	return nil
}
