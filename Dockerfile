FROM golang:1.24.4 AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y \
    gcc \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o myapp .

FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    wget \
    curl \
    zlib1g \
    unzip \
    && wget -O /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
    && chmod +x /usr/local/bin/yt-dlp \
    && curl -fsSL https://deno.land/install.sh | sh \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/myapp /app/
COPY --from=builder /app/locales /app/locales

RUN groupadd -g 1000 myuser && \
    useradd -u 1000 -g myuser -s /bin/sh myuser && \
    chown -R myuser:myuser /app

USER myuser

WORKDIR /app

ENTRYPOINT ["/app/myapp"]
