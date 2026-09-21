<div align="center">

<h1>🎵 TgMusicBot</h1>

<p>
  <b>A high-performance, low-latency Telegram audio and video streaming bot written in Go.</b>
</p>

<p>
  <a href="https://golang.org/">
    <img src="https://img.shields.io/badge/Written%20in-Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Language">
  </a>
  <a href="https://www.docker.com/">
    <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/blob/master/LICENSE">
    <img src="https://img.shields.io/badge/License-GPL%20v3-4bc51d?style=for-the-badge" alt="License">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/stargazers">
    <img src="https://img.shields.io/github/stars/AshokShau/TgMusicBot?style=for-the-badge&color=ffd700&logo=github" alt="Stars">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/network/members">
    <img src="https://img.shields.io/github/forks/AshokShau/TgMusicBot?style=for-the-badge&color=blue&logo=github" alt="Forks">
  </a>
</p>

---

<p align="center">
  TgMusicBot streams high-quality audio and up to 1080p video directly into Telegram group video chats.<br>
  Engineered with <b>Go</b>, <code>gotdbot</code> (TDLib), <code>gogram</code>, and <code>ntgcalls</code> C bindings for efficient resource usage and minimal latency.
</p>

</div>

---

## 🔥 Key Features

- **High-Performance Audio & Video**: Native Go core utilizing CGO bindings for efficient multi-track audio and video streaming in Telegram voice chats.
- **Multiple Media Sources**: Search and play directly from YouTube, Spotify, SoundCloud, Apple Music, direct HTTP/HTTPS media streams, and Telegram audio/video files.
- **Multi-Assistant Support**: Scale across up to 10 assistant userbot accounts (`STRING1` to `STRING10`) to serve multiple concurrent active voice chats.
- **Queue & Custom Playlists**: Complete queue management with track ordering, skipping, seeking, looping, and personal user playlist support.
- **Autoplay Recommendations**: Continuous music playback by automatically queuing recommended tracks when the current queue finishes.
- **Chat Admin Controls**: Per-chat authorization list, admin-only playback permissions, customizable command deletion, and settings menu.
- **Containerized Deployment**: Ready-to-use `Dockerfile` and `docker-compose.yml` preconfigured for single-command production deployment.

---

## 📋 Requirements

Before deploying, ensure you have:

1. **Linux Server** (Ubuntu 22.04 LTS or Debian 12 recommended) or a **Docker environment**.
2. **Go 1.26 or higher** (if installing manually without Docker).
3. **MongoDB Database**: Free cluster on [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) or a self-hosted instance.
4. **Telegram API Credentials**: `API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org).
5. **Telegram Bot Token**: HTTP API token generated via [@BotFather](https://t.me/BotFather).
6. **Assistant Session String**: Pyrogram or Telethon userbot session string for joining group voice chats.

---

## ⚙️ Environment Configuration

<details>
<summary><b>Click to view Environment Variables & Credentials Setup</b></summary>

<br>

Copy `sample.env` to create your configuration file:

```bash
cp sample.env .env
```

### Where to get credentials

- **`API_ID` & `API_HASH`**: Log in to [my.telegram.org](https://my.telegram.org) with your Telegram phone number, select **API development tools**, and create an application.
- **`TOKEN`**: Message [@BotFather](https://t.me/BotFather) on Telegram, send `/newbot`, follow the instructions, and copy the bot token provided.
- **`STRING1` (or `STRING`)**: Generate a Pyrogram string session for your assistant account using a session generator bot or local script.
- **`MONGO_URI`**: Register at [MongoDB Atlas](https://www.mongodb.com/cloud/atlas), create a database cluster, go to **Database Access / Network Access** to permit connections, and copy the connection string (`mongodb+srv://...`).
- **`OWNER_ID`**: Send `/id` to [@userinfobot](https://t.me/userinfobot) on Telegram to get your numeric user ID.

### Environment Variables Reference

| Variable              | Required | Default                       | Description                                                            |
|-----------------------|:--------:|-------------------------------|------------------------------------------------------------------------|
| `API_ID`              | **Yes**  | -                             | Telegram API ID from my.telegram.org.                                  |
| `API_HASH`            | **Yes**  | -                             | Telegram API Hash from my.telegram.org.                                |
| `TOKEN`               | **Yes**  | -                             | Telegram Bot Token from @BotFather.                                    |
| `OWNER_ID`            | **Yes**  | -                             | Telegram User ID of the bot owner.                                     |
| `MONGO_URI`           | **Yes**  | -                             | MongoDB connection URI string.                                         |
| `STRING1`             | **Yes**  | -                             | Assistant session string (`STRING1` to `STRING10` or `STRING`).        |
| `SESSION_TYPE`        |    No    | `pyrogram`                    | Session string format (`pyrogram` or `telethon`).                      |
| `DB_NAME`             |    No    | `Anon`                        | Database name inside MongoDB.                                          |
| `LOGGER_ID`           |    No    | `0`                           | Telegram chat/channel ID where bot startup logs and errors are sent.   |
| `DEFAULT_SERVICE`     |    No    | `youtube`                     | Default search engine for track queries (`youtube` or `spotify`).      |
| `SONG_DURATION_LIMIT` |    No    | `3600`                        | Maximum track duration allowed in seconds (default: 1 hour).           |
| `MAX_FILE_SIZE`       |    No    | `524288000`                   | Maximum file download size limit in bytes (default: 500 MB).           |
| `ENABLE_VPLAY`        |    No    | `true`                        | Enable or disable video streaming commands (`true` or `false`).        |
| `AUTO_LEAVE`          |    No    | `false`                       | Automatically leave voice chat when idle or no members remain.         |
| `COOKIES_URL`         |    No    | -                             | Comma-separated HTTP URLs pointing to raw YouTube `cookies.txt` files. |
| `SUPPORT_GROUP`       |    No    | `https://t.me/FallenSupport`  | Support group URL shown in help menus.                                 |
| `SUPPORT_CHANNEL`     |    No    | `https://t.me/FallenProjects` | Updates channel URL shown in help menus.                               |
| `START_IMG`           |    No    | (default URL)                 | Direct image URL displayed in `/start` command response.               |
| `DEVS`                |    No    | -                             | Space or comma separated list of additional developer user IDs.        |

</details>

---

## 🚀 Deployment

<details>
<summary><b>Docker Deployment (Recommended)</b></summary>

<br>

Docker isolates all dependencies (Go 1.26, FFmpeg, yt-dlp, Deno, dynamic libraries) inside a container.

#### 1. Install Docker & Docker Compose
On Ubuntu / Debian:

```bash
sudo apt update
sudo apt install -y docker.io docker-compose-v2
sudo systemctl enable --now docker
```

#### 2. Clone Repository & Configure Environment

```bash
git clone https://github.com/AshokShau/TgMusicBot.git
cd TgMusicBot
cp sample.env .env
nano .env
```

Fill in all required variables inside `.env` (`API_ID`, `API_HASH`, `TOKEN`, `OWNER_ID`, `MONGO_URI`, `STRING1`), then save and exit (`Ctrl + O`, `Enter`, `Ctrl + X`).

#### 3. Build & Run Container

Using Docker Compose:

```bash
docker compose up -d --build
```

Or using standard Docker CLI:

```bash
docker build -t tgmusic .
docker run -d --name tgmusic --env-file .env --restart unless-stopped tgmusic
```

#### 4. Container Management

- **View Logs**:
  ```bash
  docker compose logs -f
  ```
- **Stop Bot**:
  ```bash
  docker compose down
  ```
- **Restart Bot**:
  ```bash
  docker compose restart
  ```

</details>

<details>
<summary><b>Linux Setup (Ubuntu / Debian)</b></summary>

<br>

#### 1. Install System Dependencies

```bash
sudo apt update
sudo apt install -y build-essential ffmpeg curl wget unzip git
```

Install **yt-dlp**:

```bash
sudo wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

Install **Deno** (required for YouTube download challenges):

```bash
curl -fsSL https://deno.land/install.sh | sh
echo 'export DENO_INSTALL="$HOME/.deno"' >> ~/.bashrc
echo 'export PATH="$DENO_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Install **Go** (1.26 or higher required):

```bash
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. Clone Repository & Prepare Configuration

```bash
git clone https://github.com/AshokShau/TgMusicBot.git
cd TgMusicBot
cp sample.env .env
nano .env
```

Fill in your configuration settings in `.env`.

#### 3. Fetch Required Dynamic Libraries & Build

The bot uses precompiled C libraries for TDLib (`libtdjson`) and `ntgcalls`. Run the setup scripts before compiling:

```bash
# Download TDLib dynamic library
go run github.com/AshokShau/gotdbot/scripts/tools

# Download ntgcalls C libraries and headers
go run setup_ntgcalls.go

# Compile binary with CGO enabled
CGO_ENABLED=1 go build -o tgmusic main.go
```

#### 4. Run the Bot

Test run directly in terminal:

```bash
./tgmusic
```

#### 5. Background Execution

To keep the bot running after disconnecting from SSH, use `tmux` or `screen`.

##### Option A: Using `tmux`

```bash
# Start new session
tmux new -s tgmusic

# Run bot
./tgmusic
```
Detach with `Ctrl + B`, then press `D`.  
Reattach later: `tmux attach -t tgmusic`

##### Option B: Using `screen`

```bash
# Start new screen session
screen -S tgmusic

# Run bot
./tgmusic
```
Detach with `Ctrl + A`, then press `D`.  
Reattach later: `screen -r tgmusic`

</details>

---

## 🛠️ Bot Commands

<details>
<summary><b>Click to view Playback Commands</b></summary>

<br>

| Command               | Aliases | Access   | Description                                                                  |
|-----------------------|---------|----------|------------------------------------------------------------------------------|
| `/play <query/URL>`   | `/p`    | Everyone | Play audio from YouTube, Spotify, SoundCloud, direct link, or Telegram file. |
| `/mix <query/URL>`    | -       | Everyone | Create a mix of related YouTube tracks based on query or currently playing song. |
| `/vplay <query/URL>`  | `/v`    | Everyone | Stream video in group video chat.                                            |
| `/fplay <query/URL>`  | `/fp`   | Everyone | Force play audio immediately, interrupting current playback.                 |
| `/fvplay <query/URL>` | `/fvp`  | Everyone | Force play video immediately.                                                |
| `/pause`              | -       | Admin    | Pause current playback.                                                      |
| `/resume`             | -       | Admin    | Resume paused playback.                                                      |
| `/skip`               | -       | Admin    | Skip current track and play next in queue.                                   |
| `/stop`               | `/end`  | Admin    | Stop playback and clear queue.                                               |
| `/seek <seconds>`     | -       | Admin    | Jump to a timestamp in seconds.                                              |
| `/loop <0-10>`        | -       | Admin    | Repeat the current track specified number of times.                          |
| `/mute`               | -       | Admin    | Mute assistant in voice chat.                                                |
| `/unmute`             | -       | Admin    | Unmute assistant in voice chat.                                              |

</details>

<details>
<summary><b>Click to view Queue & Playlist Commands</b></summary>

<br>

| Command                  | Aliases           | Access   | Description                                   |
|--------------------------|-------------------|----------|-----------------------------------------------|
| `/queue`                 | -                 | Everyone | View current playback queue.                  |
| `/remove <index>`        | -                 | Admin    | Remove specific track from queue by position. |
| `/cplist <name>`         | `/createplaylist` | Everyone | Create a custom personal playlist.            |
| `/deleteplaylist <name>` | -                 | Everyone | Delete a personal playlist.                   |
| `/addtoplaylist`         | `/addtoplist`     | Everyone | Add track/reply message to personal playlist. |
| `/removefromplaylist`    | `/rmplist`        | Everyone | Remove track from personal playlist.          |
| `/playlistinfo <name>`   | `/plistinfo`      | Everyone | View tracks in a playlist.                    |
| `/myplaylists`           | `/myplist`        | Everyone | List all your custom playlists.               |

</details>

<details>
<summary><b>Click to view Admin & Group Setup Commands</b></summary>

<br>

| Command              | Aliases    | Access   | Description                                          |
|----------------------|------------|----------|------------------------------------------------------|
| `/join`              | `/link`    | Admin    | Invite assistant userbot to the group voice chat.    |
| `/auth <user>`       | `/addAuth` | Admin    | Grant bot admin rights in chat to a user.            |
| `/removeAuth <user>` | `/rmAuth`  | Admin    | Revoke bot admin rights from a user.                 |
| `/authList`          | `/auths`   | Everyone | List authorized users in current chat.               |
| `/settings`          | -          | Owner    | Open interactive settings menu for chat preferences. |
| `/autoplay`          | -          | Admin    | Toggle autoplay for track recommendations.           |
| `/reload`            | -          | Admin    | Refresh chat admin cache and invite links.           |

</details>

<details>
<summary><b>Click to view Owner & Developer Commands</b></summary>

<br>

| Command            | Aliases            | Access   | Description                                       |
|--------------------|--------------------|----------|---------------------------------------------------|
| `/stats`           | -                  | Devs     | Display system resource usage and bot statistics. |
| `/active_vc`       | `/av`              | Devs     | List all active voice chats across groups.        |
| `/broadcast <msg>` | `/gCast`           | Owner    | Broadcast message to served chats.                |
| `/stop_broadcast`  | `/stop_gcast`      | Owner    | Cancel active broadcast execution.                |
| `/clearass`        | `/clearAssistants` | Devs     | Reset assistant assignments.                      |
| `/leaveAll`        | -                  | Devs     | Make assistants leave all chats.                  |
| `/logger`          | -                  | Devs     | View logging channel status.                      |
| `/ping`            | -                  | Everyone | Check bot latency and uptime status.              |

</details>

---

## 🔄 Updating the Bot

When new updates are released, update your deployment using the steps below:

### Docker Deployment Update

```bash
cd TgMusicBot
git pull origin master
docker compose down
docker compose up -d --build
```

### Linux Deployment Update

```bash
cd TgMusicBot
# Stop running instance (e.g. exit tmux/screen session)

# Pull latest commits
git pull origin master

# Update Go dependencies and dynamic libraries
go mod download
go run github.com/AshokShau/gotdbot/scripts/tools
go run setup_ntgcalls.go

# Recompile binary
CGO_ENABLED=1 go build -o tgmusic main.go

# Restart process inside tmux or screen
```

---

## ❓ Troubleshooting

<details>
<summary><b>Assistant account fails to join voice chat</b></summary>

<br>

- Ensure the assistant account is not banned or restricted in the group.
- Start the Voice Chat in the Telegram group **before** running `/join` or `/play`.
- Verify that your userbot session string (`STRING1`) is active and generated from the same `API_ID` / `API_HASH`.
</details>

<details>
<summary><b>YouTube playback fails with 403 Forbidden or Sign-in errors</b></summary>

<br>

- YouTube frequently updates bot detection mechanisms.
- Export raw cookies from your browser (using extensions like *Get cookies.txt LOCALLY*).
- Upload the `cookies.txt` file to a URL or GitHub Gist (raw link) and set `COOKIES_URL` in your `.env`.
</details>

<details>
<summary><b>Build error: missing C dependencies or CGO disabled</b></summary>

<br>

- Ensure `build-essential` and `gcc` are installed on your Linux host.
- Always include `CGO_ENABLED=1` when running `go build`.
- Make sure you executed `go run github.com/AshokShau/gotdbot/scripts/tools` and `go run setup_ntgcalls.go` prior to building.
</details>

---

## 🤝 Contributing

Contributions, issues, and feature suggestions are welcome!

1. Fork the repository.
2. Create a feature branch: `git checkout -b feature/amazing-feature`.
3. Commit your changes: `git commit -m 'Add amazing feature'`.
4. Push to the branch: `git push origin feature/amazing-feature`.
5. Open a Pull Request.

---

## 📄 License

This project is licensed under the **GNU General Public License v3.0**. See the [LICENSE](../LICENSE) file for full details.

---

## 💬 Support & Updates

- **Support Group**: [Telegram Support](https://t.me/FallenSupport)
- **Updates Channel**: [Telegram Channel](https://t.me/FallenProjects)

---

## ❤️ Donate

If you find this project useful, consider supporting its development with a donation:

- **GRAM (TON) / USDT-GRAM:** `UQD8rsWDh3VD9pXVNuEbM_rIAKzV07xDhx-gzdDe0tTWGXan`
- **USDT (TRC20):** `TJWZqPK5haSE8ZdSQeWBPR5uxPSUnS8Hcq`
- **Telegram Wallet:** [@Ashokshau](https://t.me/Ashokshau)

Thank you for supporting the project!

---

<p align="center">
  Made with 🖤 by <a href="https://github.com/ashokshau">Ashok Shau</a>
</p>
