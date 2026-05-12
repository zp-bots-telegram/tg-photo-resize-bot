// Package handler wires Telegram updates to the image pipeline.
package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/semaphore"

	"github.com/zackpollard/tg-photo-resize-bot/internal/caption"
	"github.com/zackpollard/tg-photo-resize-bot/internal/exifx"
	"github.com/zackpollard/tg-photo-resize-bot/internal/pipeline"
)

const helpText = `Send me a photo as a document and I'll reply with a compressed in-chat preview plus the original resolution and EXIF info.

Supported: JPEG, PNG, HEIC. Max 20 MB.`

type Handler struct {
	log  *slog.Logger
	sem  *semaphore.Weighted
	http *http.Client
}

func New(log *slog.Logger) *Handler {
	workers := runtime.GOMAXPROCS(0)
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}
	return &Handler{
		log:  log,
		sem:  semaphore.NewWeighted(int64(workers)),
		http: &http.Client{Timeout: 60 * time.Second},
	}
}

func (h *Handler) Register(b *bot.Bot) {
	b.RegisterHandlerMatchFunc(matchHelpCommand, h.help)
	b.RegisterHandlerMatchFunc(matchDocument, h.onDocument)
}

func matchDocument(update *models.Update) bool {
	return update.Message != nil && update.Message.Document != nil
}

func matchHelpCommand(update *models.Update) bool {
	if update.Message == nil {
		return false
	}
	t := strings.TrimSpace(update.Message.Text)
	if t == "" {
		return false
	}
	// Split off any args / botname suffix.
	head := t
	if i := strings.IndexAny(t, " @"); i >= 0 {
		head = t[:i]
	}
	return head == "/start" || head == "/help"
}

func (h *Handler) help(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   helpText,
	}); err != nil {
		h.log.Warn("help send failed", "err", err)
	}
}

func (h *Handler) onDocument(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := update.Message
	doc := msg.Document
	mime := doc.MimeType

	if !supportedMime(mime) {
		// Silent on unsupported documents — Telegram groups will see a
		// lot of non-image documents and the bot shouldn't reply to all
		// of them. Matches the original Python bot's behavior.
		return
	}
	if doc.FileSize == 0 {
		h.reply(ctx, b, msg, "Couldn't determine file size — try resending.")
		return
	}
	if doc.FileSize > pipeline.MaxInputBytes {
		h.reply(ctx, b, msg, "Sorry, that's over 20 MB. Telegram won't let me download files larger than that.")
		return
	}

	progress, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          msg.Chat.ID,
		Text:            "Processing…",
		ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
	})
	if err != nil {
		h.log.Warn("progress send failed", "err", err)
		progress = nil
	}

	if err := h.sem.Acquire(ctx, 1); err != nil {
		h.editToError(ctx, b, progress, msg.Chat.ID, "Bot is shutting down — try again in a moment.")
		return
	}
	defer h.sem.Release(1)

	data, err := h.download(ctx, b, doc.FileID)
	if err != nil {
		h.log.Error("download failed", "err", err, "file_id", doc.FileID)
		h.editToError(ctx, b, progress, msg.Chat.ID, "Couldn't download the file from Telegram.")
		return
	}

	tags, exErr := exifx.Extract(data, mime)
	if exErr != nil {
		h.log.Warn("exif extract degraded", "err", exErr)
	}

	res, err := pipeline.Process(data, mime)
	if err != nil {
		h.log.Error("pipeline failed", "err", err, "mime", mime, "size", len(data))
		if errors.Is(err, pipeline.ErrUncompressible) {
			h.editToError(ctx, b, progress, msg.Chat.ID, "Image too complex to compress under 10 MB; try a smaller source.")
		} else {
			h.editToError(ctx, b, progress, msg.Chat.ID, "Sorry, I couldn't process that image.")
		}
		return
	}

	captionText := caption.Render(res.OrigWidth, res.OrigHeight, tags)

	if _, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: msg.Chat.ID,
		Photo: &models.InputFileUpload{
			Filename: "compressed.jpg",
			Data:     bytes.NewReader(res.Bytes),
		},
		Caption:         captionText,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
	}); err != nil {
		h.log.Error("send photo failed", "err", err)
		h.editToError(ctx, b, progress, msg.Chat.ID, "Processed the image but couldn't deliver it.")
		return
	}

	if progress != nil {
		if _, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    msg.Chat.ID,
			MessageID: progress.ID,
		}); err != nil {
			h.log.Warn("delete progress failed", "err", err)
		}
	}

	h.log.Info("processed",
		"chat_id", msg.Chat.ID,
		"mime", mime,
		"orig_bytes", len(data),
		"out_bytes", len(res.Bytes),
		"orig_dims", fmt.Sprintf("%dx%d", res.OrigWidth, res.OrigHeight),
		"out_dims", fmt.Sprintf("%dx%d", res.Width, res.Height),
		"pass_through", res.PassThrough,
	)
}

func (h *Handler) download(ctx context.Context, b *bot.Bot, fileID string) ([]byte, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("getFile: %w", err)
	}
	url := b.FileDownloadLink(file)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, pipeline.MaxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > pipeline.MaxInputBytes {
		return nil, fmt.Errorf("file exceeds %d byte cap", pipeline.MaxInputBytes)
	}
	return data, nil
}

func (h *Handler) reply(ctx context.Context, b *bot.Bot, msg *models.Message, text string) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:          msg.Chat.ID,
		Text:            text,
		ReplyParameters: &models.ReplyParameters{MessageID: msg.ID},
	}); err != nil {
		h.log.Warn("reply failed", "err", err)
	}
}

func (h *Handler) editToError(ctx context.Context, b *bot.Bot, progress *models.Message, chatID int64, text string) {
	if progress == nil {
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
			h.log.Warn("send fallback failed", "err", err)
		}
		return
	}
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: progress.ID,
		Text:      text,
	}); err != nil {
		h.log.Warn("edit failed", "err", err)
	}
}

func supportedMime(m string) bool {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "image/jpeg", "image/jpg", "image/png", "image/heif", "image/heic":
		return true
	}
	return false
}
