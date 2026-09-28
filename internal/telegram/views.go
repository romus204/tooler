package telegram

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/romus204/tooler/internal/models"
	"github.com/romus204/tooler/internal/store"
)

// renderCategoryView строит текст и клавиатуру для просмотра категории
// (categoryID == nil — корень каталога).
func (b *Bot) renderCategoryView(ctx context.Context, categoryID *int64) (string, tgbotapi.InlineKeyboardMarkup, error) {
	allCats, err := b.store.ListCategories(ctx)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	var title string
	var parentID *int64
	if categoryID != nil {
		cat, err := b.store.GetCategory(ctx, *categoryID)
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
		title = cat.Name
		parentID = cat.ParentID
	} else {
		title = "📂 Категории"
	}

	var children []models.Category
	for _, c := range allCats {
		if categoryID == nil {
			if c.ParentID == nil {
				children = append(children, c)
			}
			continue
		}
		if c.ParentID != nil && *c.ParentID == *categoryID {
			children = append(children, c)
		}
	}

	var tools []models.ToolWithStock
	if categoryID != nil {
		tools, err = b.store.ListTools(ctx, store.ToolFilter{CategoryID: categoryID})
		if err != nil {
			return "", tgbotapi.InlineKeyboardMarkup{}, err
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range children {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📁 "+c.Name, "cat:"+strconv.FormatInt(c.ID, 10)),
		))
	}
	for _, t := range tools {
		label := fmt.Sprintf("🔧 %s (%d %s)", t.Name, t.TotalQuantity, t.Unit)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(label, "tool:"+strconv.FormatInt(t.ID, 10)),
		))
	}
	if categoryID != nil {
		var target string
		if parentID != nil {
			target = "cat:" + strconv.FormatInt(*parentID, 10)
		} else {
			target = "cat:0"
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", target),
		))
	}

	var b2 strings.Builder
	b2.WriteString("<b>" + html.EscapeString(title) + "</b>")
	if len(children) == 0 && len(tools) == 0 {
		b2.WriteString("\n\nЗдесь пока пусто.")
	}

	return b2.String(), tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}

// renderToolView строит текст и клавиатуру для просмотра инструмента
// с быстрыми кнопками +/- на каждый размер.
func (b *Bot) renderToolView(ctx context.Context, toolID int64) (string, tgbotapi.InlineKeyboardMarkup, error) {
	detail, err := b.store.GetToolDetail(ctx, toolID)
	if err != nil {
		return "", tgbotapi.InlineKeyboardMarkup{}, err
	}

	var text strings.Builder
	text.WriteString("<b>" + html.EscapeString(detail.Name) + "</b>")
	if detail.Brand != "" {
		text.WriteString("\n" + html.EscapeString(detail.Brand))
	}
	if detail.CategoryName != "" {
		text.WriteString("\nКатегория: " + html.EscapeString(detail.CategoryName))
	}
	if detail.Description != "" {
		text.WriteString("\n" + html.EscapeString(detail.Description))
	}
	text.WriteString("\n")

	var rows [][]tgbotapi.InlineKeyboardButton
	if len(detail.Variants) == 0 {
		text.WriteString("\nРазмеров пока нет.")
	}
	for _, v := range detail.Variants {
		line := fmt.Sprintf("\n%s: <b>%d</b> %s", html.EscapeString(v.Size), v.Quantity, html.EscapeString(detail.Unit))
		if v.Location != "" {
			line += " (" + html.EscapeString(v.Location) + ")"
		}
		text.WriteString(line)

		vid := strconv.FormatInt(v.ID, 10)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➖ "+v.Size, "adj:"+vid+":-1"),
			tgbotapi.NewInlineKeyboardButtonData("➕ "+v.Size, "adj:"+vid+":1"),
		))
	}

	backTarget := "cat:0"
	if detail.CategoryID != nil {
		backTarget = "cat:" + strconv.FormatInt(*detail.CategoryID, 10)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", backTarget),
	))

	return text.String(), tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}, nil
}
