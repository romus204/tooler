package telegram

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const helpText = `<b>Учёт инструментов</b>

/categories — просмотр каталога по категориям
/search &lt;текст&gt; — поиск инструмента
/use &lt;ID&gt; &lt;кол-во&gt; [причина] — забрать (уменьшить количество)
/add &lt;ID&gt; &lt;кол-во&gt; [причина] — вернуть/пополнить
/myid — узнать свой chat ID (для настройки доступа)

ID размера указан в результатах поиска и в карточке инструмента.
Быстро менять количество можно кнопками ➖/➕ в карточке инструмента.`

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	if !b.isAllowed(ctx, chatID) {
		return
	}

	if !msg.IsCommand() {
		return
	}

	switch msg.Command() {
	case "start", "help":
		b.send(chatID, helpText, nil)

	case "myid":
		b.send(chatID, fmt.Sprintf("Ваш chat ID: <code>%d</code>", chatID), nil)

	case "categories":
		text, kb, err := b.renderCategoryView(ctx, nil)
		if err != nil {
			b.send(chatID, "Ошибка: "+html.EscapeString(err.Error()), nil)
			return
		}
		b.send(chatID, text, &kb)

	case "search":
		b.handleSearch(ctx, chatID, msg.CommandArguments())

	case "use":
		b.handleAdjustCommand(ctx, msg, -1)

	case "add":
		b.handleAdjustCommand(ctx, msg, 1)

	default:
		b.send(chatID, "Неизвестная команда. Наберите /help.", nil)
	}
}

func (b *Bot) handleSearch(ctx context.Context, chatID int64, query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		b.send(chatID, "Использование: /search текст", nil)
		return
	}
	variants, err := b.store.SearchVariants(ctx, query)
	if err != nil {
		b.send(chatID, "Ошибка поиска: "+html.EscapeString(err.Error()), nil)
		return
	}
	if len(variants) == 0 {
		b.send(chatID, "Ничего не найдено по запросу «"+html.EscapeString(query)+"».", nil)
		return
	}

	var text strings.Builder
	fmt.Fprintf(&text, "Найдено: %d\n", len(variants))
	seen := make(map[int64]bool)
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, v := range variants {
		fmt.Fprintf(&text, "\n🔧 <b>%s</b>%s — %s: %d (ID: %d)",
			html.EscapeString(v.ToolName),
			brandSuffix(v.ToolBrand),
			html.EscapeString(v.Size),
			v.Quantity,
			v.ID)
		if !seen[v.ToolID] {
			seen[v.ToolID] = true
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Открыть: "+v.ToolName, "tool:"+strconv.FormatInt(v.ToolID, 10)),
			))
		}
	}
	kb := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	b.send(chatID, text.String(), &kb)
}

func (b *Bot) handleAdjustCommand(ctx context.Context, msg *tgbotapi.Message, sign int) {
	chatID := msg.Chat.ID

	parts := strings.Fields(msg.CommandArguments())
	if len(parts) < 2 {
		b.send(chatID, "Использование: /use ID количество [причина]", nil)
		return
	}
	variantID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		b.send(chatID, "Некорректный ID размера.", nil)
		return
	}
	qty, err := strconv.Atoi(parts[1])
	if err != nil || qty <= 0 {
		b.send(chatID, "Количество должно быть положительным числом.", nil)
		return
	}
	reason := "telegram"
	if len(parts) > 2 {
		reason = strings.Join(parts[2:], " ")
	}

	v, err := b.store.GetVariantWithTool(ctx, variantID)
	if err != nil {
		b.send(chatID, "Размер с таким ID не найден.", nil)
		return
	}

	newQty, err := b.store.AdjustQuantity(ctx, variantID, sign*qty, reason, "telegram")
	if err != nil {
		b.send(chatID, "Не удалось изменить количество: "+html.EscapeString(err.Error()), nil)
		return
	}

	verb := "Использовано"
	if sign > 0 {
		verb = "Добавлено"
	}
	b.send(chatID, fmt.Sprintf(
		"%s %d — %s %s. Сейчас: <b>%d</b>.",
		verb, qty, html.EscapeString(v.ToolName), html.EscapeString(v.Size), newQty,
	), nil)
}

func brandSuffix(brand string) string {
	if brand == "" {
		return ""
	}
	return " (" + html.EscapeString(brand) + ")"
}

func (b *Bot) handleCallback(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	chatID := cq.Message.Chat.ID
	messageID := cq.Message.MessageID
	data := cq.Data

	if !b.isAllowed(ctx, chatID) {
		return
	}

	ack := tgbotapi.NewCallback(cq.ID, "")

	switch {
	case data == "noop":
		b.api.Request(ack)

	case strings.HasPrefix(data, "cat:"):
		idStr := strings.TrimPrefix(data, "cat:")
		var categoryID *int64
		if idStr != "0" {
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err == nil {
				categoryID = &id
			}
		}
		text, kb, err := b.renderCategoryView(ctx, categoryID)
		b.api.Request(ack)
		if err != nil {
			b.send(chatID, "Ошибка: "+html.EscapeString(err.Error()), nil)
			return
		}
		b.editMessage(chatID, messageID, text, kb)

	case strings.HasPrefix(data, "tool:"):
		id, err := strconv.ParseInt(strings.TrimPrefix(data, "tool:"), 10, 64)
		b.api.Request(ack)
		if err != nil {
			return
		}
		text, kb, err := b.renderToolView(ctx, id)
		if err != nil {
			b.send(chatID, "Ошибка: "+html.EscapeString(err.Error()), nil)
			return
		}
		b.editMessage(chatID, messageID, text, kb)

	case strings.HasPrefix(data, "adj:"):
		b.handleAdjustCallback(ctx, cq, data)

	default:
		b.api.Request(ack)
	}
}

func (b *Bot) handleAdjustCallback(ctx context.Context, cq *tgbotapi.CallbackQuery, data string) {
	chatID := cq.Message.Chat.ID
	messageID := cq.Message.MessageID

	parts := strings.SplitN(strings.TrimPrefix(data, "adj:"), ":", 2)
	if len(parts) != 2 {
		b.api.Request(ack(cq.ID))
		return
	}
	variantID, err1 := strconv.ParseInt(parts[0], 10, 64)
	delta, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		b.api.Request(ack(cq.ID))
		return
	}

	v, err := b.store.GetVariantWithTool(ctx, variantID)
	if err != nil {
		b.api.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Размер не найден"))
		return
	}

	newQty, err := b.store.AdjustQuantity(ctx, variantID, delta, "telegram: кнопка", "telegram")
	if err != nil {
		b.api.Request(tgbotapi.NewCallbackWithAlert(cq.ID, "Нельзя: "+err.Error()))
		return
	}

	b.api.Request(tgbotapi.NewCallback(cq.ID, fmt.Sprintf("%s %s: %d", v.ToolName, v.Size, newQty)))

	text, kb, err := b.renderToolView(ctx, v.ToolID)
	if err != nil {
		return
	}
	b.editMessage(chatID, messageID, text, kb)
}

func ack(id string) tgbotapi.CallbackConfig {
	return tgbotapi.NewCallback(id, "")
}

func (b *Bot) editMessage(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
	edit.ParseMode = tgbotapi.ModeHTML
	if _, err := b.api.Send(edit); err != nil {
		// Telegram возвращает ошибку, если текст не изменился — это не критично.
		if !strings.Contains(err.Error(), "message is not modified") {
			b.send(chatID, text, &kb)
		}
	}
}
