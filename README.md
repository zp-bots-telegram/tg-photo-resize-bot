# Telegram Photo Resize Bot

A Telegram bot that turns photo *documents* (JPEG/PNG/HEIC) into compressed
in-chat previews captioned with the original resolution and EXIF metadata
(camera, ISO, lens, shutter).

Send a photo as a file in a chat the bot is in. The bot replies with a
preview that Telegram will display inline, while the original document
stays in the chat for anyone who wants the full-quality version.

## Running

The bot is published as a Docker image and reads a single environment
variable.

```shell
docker run --rm -e TG_BOT_KEY=your_bot_api_token zackpollard/tg-photo-resize-bot
```

### Environment variables

| Name | Required | Purpose |
|------|----------|---------|
| `TG_BOT_KEY` | yes | Telegram Bot API token |

The bot uses long-polling. It does not expose any ports, write any
state, or read any configuration files.

## Commands

- `/start`, `/help` — short usage message.

## Build

```shell
docker build -t tg-photo-resize-bot .
```

### Local development

The bot uses [libvips](https://www.libvips.org/) (via
[govips](https://github.com/davidbyttow/govips)) for image processing
and [libheif](https://github.com/strukturag/libheif) for HEIC decoding.
To build outside Docker you need both system libraries.

```shell
# macOS
brew install vips libheif

# Debian / Ubuntu
sudo apt-get install libvips-dev libheif-dev pkg-config

# Build & test
go test ./...
go build ./cmd/bot
TG_BOT_KEY=... ./bot
```

## Built With

- [Go](https://go.dev/)
- [go-telegram/bot](https://github.com/go-telegram/bot) — Telegram Bot API client
- [govips](https://github.com/davidbyttow/govips) — libvips bindings
- [go-exif](https://github.com/dsoprea/go-exif) — EXIF parsing
- [go-heic-exif-extractor](https://github.com/dsoprea/go-heic-exif-extractor) — HEIC EXIF support

## Find Us

- [GitHub](https://github.com/zackpollard/tg-photo-resize-bot)
- [DockerHub](https://hub.docker.com/r/zackpollard/tg-photo-resize-bot)

## License

This project is licensed under the Unlicense License — see the [LICENSE](LICENSE) file.
