# Telegram Photo Resize Bot

A Telegram bot that turns photo *documents* (JPEG/PNG/HEIC) into compressed
in-chat previews captioned with the original resolution and EXIF metadata
(camera, ISO, lens, shutter).

Send a photo as a file in a chat the bot is in. The bot replies with a
preview that Telegram will display inline, while the original document
stays in the chat for anyone who wants the full-quality version.

## Running

The bot is published as a multi-arch (amd64 + arm64) Docker image on
GitHub Container Registry and reads a single environment variable.

```shell
docker run --rm -e TG_BOT_KEY=your_bot_api_token \
  ghcr.io/zp-bots-telegram/tg-photo-resize-bot:latest
```

### Image tags

| Tag | Meaning |
|-----|---------|
| `latest`  | Highest released semver |
| `1.2.3` / `1.2` / `1` | Specific release versions |
| `edge`    | Latest commit on `main` (unreleased) |
| `pr-<N>`  | Build from pull request #N |
| `sha-<short>` | Build from a specific commit |

### Environment variables

| Name | Required | Purpose |
|------|----------|---------|
| `TG_BOT_KEY` | yes | Telegram Bot API token |

The bot uses long-polling. It does not expose any ports, write any
state, or read any configuration files.

## Commands

- `/start`, `/help` — short usage message.

## Releases

Releases are automated by [release-please](https://github.com/googleapis/release-please).
On every merge to `main` it opens (or updates) a "Release v*x.y.z*" PR
that bumps the version and updates `CHANGELOG.md` based on
[Conventional Commits](https://www.conventionalcommits.org/). Merging
that PR creates a git tag and GitHub release, which the CI workflow
picks up and publishes as `:x.y.z` / `:x.y` / `:x` / `:latest`.

Commit message types that drive a release:
- `feat:` — minor bump
- `fix:` — patch bump
- `feat!:` / `BREAKING CHANGE:` — major bump

Other types (`refactor:`, `perf:`, `docs:`, `ci:`, `chore:`, `test:`, `build:`)
appear in the changelog but do not bump the version on their own.

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

- [GitHub](https://github.com/zp-bots-telegram/tg-photo-resize-bot)
- [Container image](https://github.com/zp-bots-telegram/tg-photo-resize-bot/pkgs/container/tg-photo-resize-bot)

## License

This project is licensed under the Unlicense License — see the [LICENSE](LICENSE) file.
