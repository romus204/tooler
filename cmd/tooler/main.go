// Command tooler запускает сервис учёта инструментов: веб-сервер и,
// опционально, Telegram-бота поверх общей базы SQLite.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/romus204/tooler/internal/config"
	"github.com/romus204/tooler/internal/db"
	"github.com/romus204/tooler/internal/store"
	"github.com/romus204/tooler/internal/telegram"
	"github.com/romus204/tooler/internal/web"
)

func main() {
	cfg := config.Load()

	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("создание директории для БД: %v", err)
		}
	}

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("открытие БД: %v", err)
	}
	defer conn.Close()

	s := store.New(conn)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(cfg.TelegramAllowedIDs) > 0 {
		for _, id := range cfg.TelegramAllowedIDs {
			if err := s.UpsertTelegramUser(ctx, id, ""); err != nil {
				log.Printf("не удалось сохранить разрешённого пользователя telegram %d: %v", id, err)
			}
		}
	}

	srv := web.NewServer(s, cfg.WebUsername, cfg.WebPassword)
	httpServer := &http.Server{
		Addr:    cfg.WebAddr,
		Handler: srv.Router(),
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("веб-сервер слушает %s", cfg.WebAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("веб-сервер остановлен с ошибкой: %v", err)
		}
	}()

	if cfg.TelegramToken != "" {
		bot, err := telegram.New(cfg.TelegramToken, cfg.TelegramAllowedIDs, s)
		if err != nil {
			log.Printf("не удалось запустить telegram-бота: %v", err)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
					log.Printf("telegram-бот остановлен с ошибкой: %v", err)
				}
			}()
		}
	} else {
		log.Println("TOOLER_TELEGRAM_TOKEN не задан — telegram-бот не запущен")
	}

	<-ctx.Done()
	log.Println("получен сигнал остановки, завершаем работу...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка остановки веб-сервера: %v", err)
	}

	wg.Wait()
	log.Println("остановлено")
}
