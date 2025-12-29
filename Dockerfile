FROM golang:1.25.4 AS builder

WORKDIR /app

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    gcc \
    zlib1g-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go generate

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o main .

FROM golang:1.25.4 AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    wget \
    unzip \
    curl \
    lsb-release \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*


RUN wget -O /usr/local/bin/yt-dlp \
    https://github.com/yt-dlp/yt-dlp-nightly-builds/releases/latest/download/yt-dlp_linux \
    && chmod +x /usr/local/bin/yt-dlp \
    && curl -fsSL https://deno.land/install.sh | sh \
    && export DENO_INSTALL="/root/.deno" \
    && export PATH="$DENO_INSTALL/bin:$PATH" \
    && ln -sf /root/.deno/bin/deno /usr/local/bin/deno

ENV DENO_INSTALL="/root/.deno"
ENV PATH="${DENO_INSTALL}/bin:${PATH}"

WORKDIR /root/
COPY --from=builder /app/main .

ENTRYPOINT ["./main"]
