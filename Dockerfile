FROM golang:1.26-alpine3.23 AS builder

WORKDIR /app

RUN apk add --no-cache build-base zlib-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate .
RUN go run setup_ntgcalls.go

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .

FROM alpine:3.23 AS runtime

RUN apk add --no-cache \
    ca-certificates \
    deno \
    ffmpeg \
    libstdc++ \
    openssl \
    wget \
    zlib

RUN case "$(apk --print-arch)" in \
      x86_64) yt_dlp_asset=yt-dlp_musllinux ;; \
      aarch64) yt_dlp_asset=yt-dlp_musllinux_aarch64 ;; \
      *) echo "Unsupported architecture" >&2; exit 1 ;; \
    esac && \
    wget -q -O /usr/local/bin/yt-dlp \
      "https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/$yt_dlp_asset" && \
    chmod +x /usr/local/bin/yt-dlp

RUN addgroup -S app && adduser -S -G app -h /home/app app

ENV HOME="/home/app"

COPY --from=builder --chown=app:app /app/main /usr/local/bin/app
COPY --from=builder --chown=app:app /app/libtdjson.so.* /home/app/

USER app

WORKDIR /home/app
ENTRYPOINT ["app"]
