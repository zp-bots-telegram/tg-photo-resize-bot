FROM golang:1.25-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends \
      pkg-config \
      libvips-dev \
      libheif-dev \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" \
      -o /out/bot ./cmd/bot

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      libvips42 \
      libheif1 \
      ca-certificates \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -r -u 10001 -g nogroup bot
COPY --from=build /out/bot /usr/local/bin/bot
USER bot
ENTRYPOINT ["/usr/local/bin/bot"]
