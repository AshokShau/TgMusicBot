FROM golang:1.24.4 AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y \
    git \
    gcc \
    zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum setup_ntgcalls.go ./
RUN go mod download

COPY . .

RUN go generate
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o myapp .

FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    ffmpeg \
    wget \
    zlib1g \
    && wget -O /usr/local/bin/yt-dlp \
       https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux \
    && chmod +x /usr/local/bin/yt-dlp \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/myapp /app/
COPY --from=builder /app/pkg/lang/locale /app/pkg/lang/locale

RUN chmod +x /app/myapp

WORKDIR /app

ENTRYPOINT ["/app/myapp"]
VOLUME ["/app"]
