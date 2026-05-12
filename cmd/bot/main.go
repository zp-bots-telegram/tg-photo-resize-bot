package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
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

	cfg := handler.Config{}
	if s := os.Getenv("TG_MAX_INPUT_BYTES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			log.Error("TG_MAX_INPUT_BYTES must be a positive integer", "value", s)
			os.Exit(1)
		}
		cfg.MaxInputBytes = n
	}
	h := handler.New(log, cfg)

	var opts []bot.Option
	if apiURL := os.Getenv("TG_API_URL"); apiURL != "" {
		log.Info("using custom Bot API server", "url", apiURL)
		opts = append(opts, bot.WithServerURL(apiURL))
	}
	b, err := bot.New(token, opts...)
	if err != nil {
		log.Error("bot.New failed", "err", err)
		os.Exit(1)
	}
	h.Register(b)

	log.Info("bot started")
	b.Start(ctx)
	log.Info("bot stopped")
}
