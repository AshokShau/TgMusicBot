FROM golang:1.25-bookworm AS builder

WORKDIR /src

RUN apt-get update && apt-get install -y --no-install-recommends gcc zlib1g-dev && \
    rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o app .


FROM debian:12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    zlib1g \
    wget \
    curl \
    unzip && \
    rm -rf /var/lib/apt/lists/*


RUN wget -q -O /usr/local/bin/yt-dlp \
      https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux && \
    chmod +x /usr/local/bin/yt-dlp


RUN curl -fsSL https://deno.land/install.sh | DENO_INSTALL="/home/app/.deno" sh
ENV PATH="/home/app/.deno/bin:${PATH}"

RUN useradd -m -u 1000 app

COPY --from=builder /src/app /app/app
COPY --from=builder /src/config /app/config

RUN chmod +x /app/app && \
    chown -R app:app /app

USER app

WORKDIR /app
ENTRYPOINT ["/app/app"]
