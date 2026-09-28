package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/romus204/tooler/internal/models"
)

func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, parent_id, name, created_at FROM categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []models.Category
	for rows.Next() {
		var c models.Category
		var parent sql.NullInt64
		if err := rows.Scan(&c.ID, &parent, &c.Name, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.ParentID = ptrFromNull(parent)
		out = append(out, c)
	}
	return out, rows.Err()
}

// CategoryTree строит дерево категорий из плоского списка.
func (s *Store) CategoryTree(ctx context.Context) ([]*models.CategoryNode, error) {
	flat, err := s.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make(map[int64]*models.CategoryNode, len(flat))
	for _, c := range flat {
		nodes[c.ID] = &models.CategoryNode{Category: c}
	}

	var roots []*models.CategoryNode
	for _, c := range flat {
		node := nodes[c.ID]
		if c.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[*c.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots, nil
}

func (s *Store) GetCategory(ctx context.Context, id int64) (models.Category, error) {
	var c models.Category
	var parent sql.NullInt64
	row := s.db.QueryRowContext(ctx, `SELECT id, parent_id, name, created_at FROM categories WHERE id = ?`, id)
	if err := row.Scan(&c.ID, &parent, &c.Name, &c.CreatedAt); err != nil {
		return c, mapNotFound(err)
	}
	c.ParentID = ptrFromNull(parent)
	return c, nil
}

func (s *Store) CreateCategory(ctx context.Context, name string, parentID *int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO categories (parent_id, name) VALUES (?, ?)`, nullInt64(parentID), name)
	if err != nil {
		return 0, fmt.Errorf("create category: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateCategory(ctx context.Context, id int64, name string, parentID *int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE categories SET name = ?, parent_id = ? WHERE id = ?`, name, nullInt64(parentID), id)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	return nil
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}
