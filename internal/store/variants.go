package store

import (
	"context"
	"fmt"

	"github.com/romus204/tooler/internal/models"
)

func (s *Store) ListVariantsForTool(ctx context.Context, toolID int64) ([]models.ToolVariant, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tool_id, size, sku, quantity, location, notes, created_at, updated_at
		 FROM tool_variants WHERE tool_id = ? ORDER BY size`, toolID)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var out []models.ToolVariant
	for rows.Next() {
		var v models.ToolVariant
		if err := rows.Scan(&v.ID, &v.ToolID, &v.Size, &v.SKU, &v.Quantity, &v.Location, &v.Notes, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetVariant(ctx context.Context, id int64) (models.ToolVariant, error) {
	var v models.ToolVariant
	row := s.db.QueryRowContext(ctx,
		`SELECT id, tool_id, size, sku, quantity, location, notes, created_at, updated_at
		 FROM tool_variants WHERE id = ?`, id)
	if err := row.Scan(&v.ID, &v.ToolID, &v.Size, &v.SKU, &v.Quantity, &v.Location, &v.Notes, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return v, mapNotFound(err)
	}
	return v, nil
}

// GetVariantWithTool возвращает размер вместе с данными инструмента и категории.
func (s *Store) GetVariantWithTool(ctx context.Context, id int64) (models.VariantWithTool, error) {
	var v models.VariantWithTool
	row := s.db.QueryRowContext(ctx, `
		SELECT
			tv.id, tv.tool_id, tv.size, tv.sku, tv.quantity, tv.location, tv.notes, tv.created_at, tv.updated_at,
			t.name, t.brand, COALESCE(c.name, '')
		FROM tool_variants tv
		JOIN tools t ON t.id = tv.tool_id
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE tv.id = ?`, id)
	if err := row.Scan(
		&v.ID, &v.ToolID, &v.Size, &v.SKU, &v.Quantity, &v.Location, &v.Notes, &v.CreatedAt, &v.UpdatedAt,
		&v.ToolName, &v.ToolBrand, &v.CategoryName,
	); err != nil {
		return v, mapNotFound(err)
	}
	return v, nil
}

func (s *Store) CreateVariant(ctx context.Context, v models.ToolVariant) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tool_variants (tool_id, size, sku, quantity, location, notes) VALUES (?, ?, ?, ?, ?, ?)`,
		v.ToolID, v.Size, v.SKU, v.Quantity, v.Location, v.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("create variant: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if v.Quantity != 0 {
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO stock_movements (variant_id, delta, reason, source) VALUES (?, ?, ?, ?)`,
			id, v.Quantity, "начальное количество", "web",
		); err != nil {
			return id, fmt.Errorf("log initial movement: %w", err)
		}
	}
	return id, nil
}

func (s *Store) UpdateVariant(ctx context.Context, v models.ToolVariant) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tool_variants SET size = ?, sku = ?, location = ?, notes = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		v.Size, v.SKU, v.Location, v.Notes, v.ID,
	)
	if err != nil {
		return fmt.Errorf("update variant: %w", err)
	}
	return nil
}

func (s *Store) DeleteVariant(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tool_variants WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete variant: %w", err)
	}
	return nil
}

// AdjustQuantity меняет количество на delta (может быть отрицательным) и
// пишет запись в историю. Не позволяет уйти в минус.
func (s *Store) AdjustQuantity(ctx context.Context, variantID int64, delta int, reason, source string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var current int
	if err := tx.QueryRowContext(ctx, `SELECT quantity FROM tool_variants WHERE id = ?`, variantID).Scan(&current); err != nil {
		return 0, mapNotFound(err)
	}

	newQty := current + delta
	if newQty < 0 {
		return 0, fmt.Errorf("недостаточно: сейчас %d, запрошено %d", current, delta)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tool_variants SET quantity = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newQty, variantID); err != nil {
		return 0, fmt.Errorf("update quantity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO stock_movements (variant_id, delta, reason, source) VALUES (?, ?, ?, ?)`, variantID, delta, reason, source); err != nil {
		return 0, fmt.Errorf("log movement: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return newQty, nil
}

// SearchVariants ищет по названию/бренду/размеру/коду инструмента.
func (s *Store) SearchVariants(ctx context.Context, query string) ([]models.VariantWithTool, error) {
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			tv.id, tv.tool_id, tv.size, tv.sku, tv.quantity, tv.location, tv.notes, tv.created_at, tv.updated_at,
			t.name, t.brand, COALESCE(c.name, '')
		FROM tool_variants tv
		JOIN tools t ON t.id = tv.tool_id
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.name LIKE ? OR t.brand LIKE ? OR tv.size LIKE ? OR tv.sku LIKE ?
		ORDER BY t.name, tv.size
		LIMIT 30`, like, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("search variants: %w", err)
	}
	defer rows.Close()
	return scanVariantsWithTool(rows)
}

func scanVariantsWithTool(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]models.VariantWithTool, error) {
	var out []models.VariantWithTool
	for rows.Next() {
		var v models.VariantWithTool
		if err := rows.Scan(
			&v.ID, &v.ToolID, &v.Size, &v.SKU, &v.Quantity, &v.Location, &v.Notes, &v.CreatedAt, &v.UpdatedAt,
			&v.ToolName, &v.ToolBrand, &v.CategoryName,
		); err != nil {
			return nil, fmt.Errorf("scan variant with tool: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
