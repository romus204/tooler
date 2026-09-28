package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// UpsertTelegramUser добавляет разрешённого пользователя Telegram (идемпотентно).
func (s *Store) UpsertTelegramUser(ctx context.Context, chatID int64, username string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO telegram_users (chat_id, username) VALUES (?, ?)
		 ON CONFLICT(chat_id) DO UPDATE SET username = excluded.username`,
		chatID, username,
	)
	if err != nil {
		return fmt.Errorf("upsert telegram user: %w", err)
	}
	return nil
}

func (s *Store) IsTelegramUserAllowed(ctx context.Context, chatID int64) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM telegram_users WHERE chat_id = ?`, chatID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
