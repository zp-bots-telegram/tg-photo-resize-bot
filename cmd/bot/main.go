package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/go-telegram/bot"

	"github.com/zackpollard/tg-photo-resize-bot/internal/handler"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	token := os.Getenv("TG_BOT_KEY")
	if token == "" {
		log.Error("TG_BOT_KEY is not set")
		os.Exit(1)
	}

	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(&vips.Config{
		ConcurrencyLevel: 1,
		MaxCacheMem:      64 * 1024 * 1024,
		MaxCacheSize:     0,
		MaxCacheFiles:    0,
	})
	defer vips.Shutdown()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	h := handler.New(log)

	b, err := bot.New(token)
	if err != nil {
		log.Error("bot.New failed", "err", err)
		os.Exit(1)
	}
	h.Register(b)

	log.Info("bot started")
	b.Start(ctx)
	log.Info("bot stopped")
}
