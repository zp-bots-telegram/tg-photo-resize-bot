package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

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

	debug := os.Getenv("TG_DEBUG") == "1"

	var opts []bot.Option
	if apiURL := os.Getenv("TG_API_URL"); apiURL != "" {
		log.Info("using custom Bot API server", "url", apiURL)
		opts = append(opts, bot.WithServerURL(apiURL))
	}
	if debug {
		opts = append(opts,
			bot.WithDebug(),
			bot.WithDebugHandler(func(format string, args ...any) {
				log.Info("bot debug", "msg", strings.TrimRight(fmt.Sprintf(format, args...), "\n"))
			}),
			bot.WithHTTPClient(5*time.Minute, &http.Client{
				Transport: &dumpTransport{
					inner: http.DefaultTransport,
					log:   log,
					token: token,
				},
			}),
		)
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

// dumpTransport logs every HTTP request and response the bot library
// makes. Enabled by TG_DEBUG=1. Bodies are dumped verbatim; the bot
// token is masked so log lines stay shareable.
type dumpTransport struct {
	inner http.RoundTripper
	log   *slog.Logger
	token string
}

func (d *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqDump, _ := httputil.DumpRequestOut(req, true)
	d.log.Info("http request", "dump", d.mask(string(reqDump)))

	resp, err := d.inner.RoundTrip(req)
	if err != nil {
		d.log.Error("http transport error", "err", err)
		return resp, err
	}
	// DumpResponse consumes the body; replace it so the caller can read it.
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	respDump, _ := httputil.DumpResponse(resp, false)
	d.log.Info("http response", "status", resp.Status, "headers", d.mask(string(respDump)), "body", d.mask(string(body)))
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func (d *dumpTransport) mask(s string) string {
	if d.token == "" {
		return s
	}
	return strings.ReplaceAll(s, d.token, "<TOKEN>")
}
