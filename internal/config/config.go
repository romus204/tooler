// Package config читает настройки приложения из переменных окружения.
package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Путь к файлу базы данных SQLite.
	DBPath string
	// Адрес, на котором слушает веб-сервер, например ":8080".
	WebAddr string
	// Логин/пароль для базовой HTTP-аутентификации веб-морды.
	// Если оба пустые — аутентификация отключена.
	WebUsername string
	WebPassword string

	// Токен Telegram-бота. Если пустой — бот не запускается.
	TelegramToken string
	// Chat ID пользователей, которым разрешено изменять остатки через бота.
	TelegramAllowedIDs []int64
}

func Load() Config {
	cfg := Config{
		DBPath:        getEnv("TOOLER_DB_PATH", "data/tooler.db"),
		WebAddr:       getEnv("TOOLER_WEB_ADDR", ":8080"),
		WebUsername:   os.Getenv("TOOLER_WEB_USERNAME"),
		WebPassword:   os.Getenv("TOOLER_WEB_PASSWORD"),
		TelegramToken: os.Getenv("TOOLER_TELEGRAM_TOKEN"),
	}

	if raw := os.Getenv("TOOLER_TELEGRAM_ALLOWED_IDS"); raw != "" {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if id, err := strconv.ParseInt(part, 10, 64); err == nil {
				cfg.TelegramAllowedIDs = append(cfg.TelegramAllowedIDs, id)
			}
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
