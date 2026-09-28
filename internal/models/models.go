// Package models содержит доменные типы каталога инструментов.
package models

import "time"

type Category struct {
	ID        int64
	ParentID  *int64
	Name      string
	CreatedAt time.Time
}

// CategoryNode — узел дерева категорий для отображения в веб-морде.
type CategoryNode struct {
	Category
	Children []*CategoryNode
}

type Tool struct {
	ID          int64
	CategoryID  *int64
	Name        string
	Brand       string
	Description string
	Unit        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ToolVariant struct {
	ID        int64
	ToolID    int64
	Size      string
	SKU       string
	Quantity  int
	Location  string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type StockMovement struct {
	ID        int64
	VariantID int64
	Delta     int
	Reason    string
	Source    string
	CreatedAt time.Time
}

// ToolWithStock — инструмент со сводным количеством, используется в списках.
type ToolWithStock struct {
	Tool
	CategoryName  string
	VariantCount  int
	TotalQuantity int
}

// ToolDetail — инструмент со всеми размерами, для карточки инструмента.
type ToolDetail struct {
	Tool
	CategoryName string
	Variants     []ToolVariant
}

// VariantWithTool — размер вместе с данными родительского инструмента,
// используется в поиске.
type VariantWithTool struct {
	ToolVariant
	ToolName     string
	ToolBrand    string
	CategoryName string
}
