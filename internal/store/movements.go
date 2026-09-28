package store

import (
	"context"
	"fmt"

	"github.com/romus204/tooler/internal/models"
)

func (s *Store) ListMovementsForVariant(ctx context.Context, variantID int64, limit int) ([]models.StockMovement, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, variant_id, delta, reason, source, created_at FROM stock_movements WHERE variant_id = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		variantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var out []models.StockMovement
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.VariantID, &m.Delta, &m.Reason, &m.Source, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan movement: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
