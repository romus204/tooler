// Package telegram реализует Telegram-бота для работы с каталогом инструментов.
package telegram

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/romus204/tooler/internal/store"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	store   *store.Store
	allowed map[int64]bool
}

// New создаёт бота. allowedIDs — chat ID пользователей, которым разрешено
// изменять остатки (кроме тех, что уже сохранены в БД через telegram_users).
func New(token string, allowedIDs []int64, s *store.Store) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	allowed := make(map[int64]bool, len(allowedIDs))
	for _, id := range allowedIDs {
		allowed[id] = true
	}
	return &Bot{api: api, store: s, allowed: allowed}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	log.Printf("telegram bot started as @%s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := b.api.GetUpdatesChan(u)
	defer b.api.StopReceivingUpdates()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("telegram: recovered panic: %v", r)
		}
	}()

	switch {
	case update.Message != nil:
		b.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		b.handleCallback(ctx, update.CallbackQuery)
	}
}

func (b *Bot) isAllowed(ctx context.Context, chatID int64) bool {
	if b.allowed[chatID] {
		return true
	}
	ok, err := b.store.IsTelegramUserAllowed(ctx, chatID)
	if err != nil {
		log.Printf("telegram: check allowed user: %v", err)
		return false
	}
	return ok
}

func (b *Bot) send(chatID int64, text string, kb *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("telegram: send message: %v", err)
	}
}
