# Installation and Deployment Guide

This guide provides detailed instructions for installing, configuring, deploying, updating, and troubleshooting TgMusicBot.

## Requirements

Ensure your system meets the following prerequisites before proceeding with manual installation:

- **Operating System**: Linux (Ubuntu 22.04+ recommended), macOS, or Windows
- **Go**: Version 1.26 or higher
- **C Compiler**: GCC / Clang (required for CGO bindings to `ntgcalls`)
- **FFmpeg**: Installed and available in your system `PATH`
- **yt-dlp**: Installed and available in your system `PATH`
- **Deno**: Installed and available in system `PATH` (used for JavaScript challenges during downloads)
- **MongoDB**: Active connection string (MongoDB Atlas or local MongoDB server)
- **Git**: Installed for cloning the repository

## Telegram API Credentials & Database

### Telegram API Keys

1. Visit [my.telegram.org](https://my.telegram.org) and log in with your Telegram phone number.
2. Go to **API development tools**.
3. Create a new application to obtain your `API_ID` and `API_HASH`.

### Bot Token

1. Open Telegram and start a chat with [@BotFather](https://t.me/BotFather).
2. Send `/newbot` and follow the prompts to create your bot.
3. Save the HTTP API bot token (`TOKEN`).

### Assistant Session String

TgMusicBot requires an assistant Telegram user account (userbot) to join group voice chats and stream audio/video.

Generate a session string using a Pyrogram or Telethon session string generator. Save this string as `STRING` (or `STRING1` through `STRING10` if using multiple assistant accounts).

### MongoDB Database Setup

1. Create a cluster on [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) or set up a local MongoDB instance.
2. Obtain your connection URI (e.g., `mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority`).
3. Ensure network access rules allow your server IP to connect to the database.

## Environment Configuration

Clone the repository and prepare the configuration file:

```bash
git clone https://github.com/AshokShau/TgMusicBot.git
cd TgMusicBot
cp sample.env .env
```

Open `.env` in a text editor and fill in your values.

### Environment Variables Reference

| Variable              | Required | Default                       | Description                                                             |
|-----------------------|----------|-------------------------------|-------------------------------------------------------------------------|
| `API_ID`              | Yes      | -                             | Telegram API ID from my.telegram.org.                                   |
| `API_HASH`            | Yes      | -                             | Telegram API Hash from my.telegram.org.                                 |
| `TOKEN`               | Yes      | -                             | Telegram Bot Token from @BotFather.                                     |
| `MONGO_URI`           | Yes      | -                             | MongoDB connection URI.                                                 |
| `OWNER_ID`            | Yes      | -                             | Telegram user ID of the bot owner.                                      |
| `STRING`              | Yes*     | -                             | Userbot session string (`STRING`, or `STRING1` to `STRING10`).          |
| `SESSION_TYPE`        | No       | `pyrogram`                    | Session string format: `pyrogram` or `telethon`.                        |
| `DL_BOT_TOKEN`        | No       | -                             | Optional separate Telegram Bot Token for downloader client operations.  |
| `DB_NAME`             | No       | `Anon`                        | Database name in MongoDB.                                               |
| `LOGGER_ID`           | No       | `0`                           | Chat ID of the log group for status messages and errors.                |
| `API_URL`             | No       | `https://api.onegrab.fun`     | API endpoint for external download resolvers.                           |
| `API_KEY`             | No       | -                             | Optional API key for external downloader service.                       |
| `DEFAULT_SERVICE`     | No       | `youtube`                     | Default search and playback provider (`youtube` or `spotify`).          |
| `SONG_DURATION_LIMIT` | No       | `3600`                        | Maximum song duration in seconds allowed for playback.                  |
| `MAX_FILE_SIZE`       | No       | `524288000`                   | Maximum allowed file size for downloads in bytes (500 MB).              |
| `DOWNLOADS_DIR`       | No       | `database`                    | Local directory for storing temporary downloaded files.                 |
| `COOKIES_URL`         | No       | -                             | Comma-separated list of URLs pointing to YouTube cookies files.         |
| `SUPPORT_GROUP`       | No       | `https://t.me/FallenSupport`  | Telegram link for support group.                                        |
| `SUPPORT_CHANNEL`     | No       | `https://t.me/FallenProjects` | Telegram link for updates channel.                                      |
| `START_IMG`           | No       | `https://i.pinimg.com/...`    | Image URL displayed in the `/start` command response.                   |
| `PORT`                | No       | `6060`                        | HTTP server port for health checks and pprof profiling.                 |
| `AUTO_LEAVE`          | No       | `false`                       | Automatically leave voice chat when alone or idle.                      |
| `ENABLE_VPLAY`        | No       | `true`                        | Enable or disable video playback commands.                              |
| `DEVS`                | No       | -                             | Space-separated or comma-separated list of developer Telegram user IDs. |

---

## Local Installation

### System Dependencies

#### Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y build-essential ffmpeg curl wget unzip git
```

Install yt-dlp:

```bash
sudo wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

Install Deno:

```bash
curl -fsSL https://deno.land/install.sh | sh
echo 'export DENO_INSTALL="$HOME/.deno"' >> ~/.bashrc
echo 'export PATH="$DENO_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### macOS

```bash
brew install go ffmpeg yt-dlp deno
```

#### Windows

1. Install Go 1.26+ from [go.dev](https://go.dev/dl/).
2. Install GCC (via MinGW-w64 or w64devkit) and add it to your System PATH.
3. Install FFmpeg, yt-dlp, and Deno, ensuring all executables are added to your System PATH.

---

### Fetch Required Libraries

TgMusicBot relies on TDLib C headers (`libtdjson`) and prebuilt static `ntgcalls` C libraries. Run the provided helper scripts before building:

```bash
go run github.com/AshokShau/gotdbot/scripts/tools
go run setup_ntgcalls.go
```

The scripts will download and place `libtdjson` in the project directory and `ntgcalls` C libraries into `internal/calls/` and `ntgcalls/`.

---

### Build and Run

Build the application with CGO enabled:

```bash
CGO_ENABLED=1 go build -o tgmusic main.go
```

Run the compiled executable:

```bash
./tgmusic
```

---

## Background Execution

To keep the bot running after closing your SSH session, use `screen` or `tmux`.

### Using `screen`

Start a new screen session:

```bash
screen -S tgmusic
```

Run the bot:

```bash
./tgmusic
```

Detach from the screen session by pressing `Ctrl + A`, then `D`.

Reattach to the session later:

```bash
screen -r tgmusic
```

### Using `tmux`

Start a new tmux session:

```bash
tmux new -s tgmusic
```

Run the bot:

```bash
./tgmusic
```

Detach from tmux by pressing `Ctrl + B`, then `D`.

Reattach to the session:

```bash
tmux attach -t tgmusic
```

---

## Production Deployment with systemd

For Linux servers, configuring a systemd service ensures automatic restarts upon system reboots or unexpected crashes.

1. Create a systemd service file:

   ```bash
   sudo nano /etc/systemd/system/tgmusic.service
   ```

2. Add the following configuration (update paths and user to match your environment):

   ```ini
   [Unit]
   Description=TgMusicBot Service
   After=network.target

   [Service]
   Type=simple
   User=ubuntu
   WorkingDirectory=/home/ubuntu/TgMusicBot
   ExecStart=/home/ubuntu/TgMusicBot/tgmusic
   Restart=always
   RestartSec=5
   EnvironmentFile=/home/ubuntu/TgMusicBot/.env

   [Install]
   WantedBy=multi-user.target
   ```

3. Reload systemd, enable, and start the service:

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable tgmusic
   sudo systemctl start tgmusic
   ```

4. Manage the service:

    - Check status: `sudo systemctl status tgmusic`
    - View logs: `journalctl -u tgmusic -f`
    - Restart service: `sudo systemctl restart tgmusic`
    - Stop service: `sudo systemctl stop tgmusic`

---

## Docker Deployment

TgMusicBot includes a Dockerfile and `docker-compose.yml` preconfigured with Cloudflare WARP routing for bypass performance.

### Using Docker Compose (Recommended)

1. Ensure Docker and Docker Compose are installed on your system.
2. Prepare your `.env` file in the project root directory.
3. Start the container in detached mode:

   ```bash
   docker-compose up -d --build
   ```

4. View logs:

   ```bash
   docker-compose logs -f
   ```

5. Stop the container:

   ```bash
   docker-compose down
   ```

### Using Docker CLI

Build the Docker image:

```bash
docker build -t tgmusic .
```

Run the container:

```bash
docker run -d --name docker build -t tgmusic . --env-file .env --restart unless-stopped tgmusic
```

---

## Updating the Bot

To update your deployment to the latest commit:

1. Stop the running bot process or service:

   ```bash
   sudo systemctl stop tgmusic
   # or: docker-compose down
   ```

2. Pull the latest code changes:

   ```bash
   git pull origin master
   ```

3. Update dependencies and dynamic libraries:

   ```bash
   go mod download
   go run github.com/AshokShau/gotdbot/scripts/tools
   go run setup_ntgcalls.go
   ```

4. Rebuild the executable:

   ```bash
   CGO_ENABLED=1 go build -o tgmusic main.go
   ```

5. Restart the bot process or service:

   ```bash
   sudo systemctl start tgmusic
   # or: docker-compose up -d --build
   ```

---
