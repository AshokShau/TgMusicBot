<h1 align="center">🎵 TGMusic Bot (Go)</h1>

<p align="center">
  <a href="https://golang.org/">
    <img src="https://img.shields.io/badge/Written%20in-Go-blue?style=for-the-badge&logo=go" alt="Written in Go">
  </a>
  <a href="https://docs.docker.com/">
    <img src="https://img.shields.io/badge/Docker-Enabled-blue?style=for-the-badge&logo=docker" alt="Docker">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-GPL%20v3-green?style=for-the-badge" alt="License">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/stargazers">
    <img src="https://img.shields.io/github/stars/AshokShau/TgMusicBot?style=for-the-badge&color=ffd700&logo=github" alt="Stars">
  </a>
</p>

<p align="center">
  A high-performance, feature-rich Telegram Music Bot written in <b>Go</b>. <br>
  Built with <code>gogram</code>, <code>ntgcalls</code>, and <code>mongo-driver</code>.
</p>

<p align="center">
    <a href="https://heroku.com/deploy?template=https://github.com/AshokShau/TgMusicBot">
        <img src="https://www.herokucdn.com/deploy/button.svg" alt="Deploy">
    </a>
</p>

---

## ✨ Features

- 🚀 **High Performance**: Written in Go for efficiency and speed.
- 📹 **Video & Audio**: Supports playing both video and audio streams.
- 🔗 **Multi-Source**: YouTube, Direct Links, M3U8, etc.
- 🎛 **Advanced Control**: Seek, Pause, Resume, Mute, Volume Control.
- 📋 **Playlist Management**: Queue system with skip, loop, and shuffle.
- 🌐 **Multi-Language**: Easy to localize for different regions.
- 🐳 **Docker Ready**: Easy deployment with Docker.

---

## 🛠️ Installation & Setup

<details>
<summary><b>Click to reveal Local Setup Instructions</b></summary>

### Prerequisites
- **Go 1.25+**
- **FFmpeg**

### Steps

1. **Clone the repository**
   ```bash
   git clone https://github.com/AshokShau/TgMusicBot.git
   cd TgMusicBot
   ```

2. **Setup `ntgcalls` & `gotdbot`**
   Download the required C++ libraries.
   ```bash
   go run setup_ntgcalls.go
   ```

   ```bash
   go run github.com/AshokShau/gotdbot/scripts/tools@latest
   ```

3. **Configure Environment**
   Copy the sample file and edit it.
   ```bash
   cp sample.env .env
   vi .env
   ```

4. **Build and Run**
   ```bash
   go build -o app .
   ./app
   ```

5. **Run in Background**
   - **Quick Start**: Use `screen` or `tmux`.
   - **Production**: Use `systemd`.
     See [installation.md](installation.md#running-in-background) for details.

For detailed instructions, see [installation.md](installation.md).

</details>

<details>
<summary><b>Click to reveal Docker Setup Instructions</b></summary>

1. **Clone and Config**
   ```bash
   git clone https://github.com/AshokShau/TgMusicBot.git
   cd TgMusicBot
   cp sample.env .env
   # Edit .env with your credentials
   ```

2. **Run with Docker Compose**
   ```bash
   docker-compose up -d --build
   ```

</details>

<details>
<summary><b>Click to reveal Heroku Setup Instructions</b></summary>

1. Click the **Deploy to Heroku** button above.
2. Fill in the required environment variables (`API_ID`, `API_HASH`, `STRING1`, etc.).
3. Deploy the app.
4. **Important**: Enable the `worker` dyno and disable the `web` dyno in the Resources tab.

</details>

---

## ⚙️ Configuration

The bot is configured via environment variables. See `sample.env` for all options.

<details>
<summary><b>Click to view Environment Variables</b></summary>

| Variable              | Description                   | Required |
|:----------------------|:------------------------------|:--------:|
| `API_ID`              | Telegram API ID               |    ✅     |
| `API_HASH`            | Telegram API Hash             |    ✅     |
| `TOKEN`               | Bot Token from @BotFather     |    ✅     |
| `STRING1`             | Pyrogram V2 Session String    |    ✅     |
| `MONGO_URI`           | MongoDB Connection URI        |    ✅     |
| `OWNER_ID`            | Telegram User ID of the owner |    ✅     |
| `LOGGER_ID`           | Group chat ID for logs        |    ❌     |
| `SONG_DURATION_LIMIT` | Max song duration in seconds  |    ❌     |
| `API_KEY`             | Your API key                  |    ❌     |
| `COOKIES_URL`         | Cookies URL for the bot       |    ❌     |

</details>

---

## 🤖 Commands

<details>
<summary><b>Click to view Admin Commands</b></summary>

- `/play <query|url>` - Play a song or video.
- `/vplay <query|url>` - Force video play.
- `/skip` - Skip the current track.
- `/pause` - Pause playback.
- `/resume` - Resume playback.
- `/end` - Stop playback and clear queue.
- `/mute` - Mute the assistant.
- `/unmute` - Unmute the assistant.
- `/auth` - Authorize a user to use the bot.
- `/unauth` - Revoke authorization.
- `/settings` - Configure bot settings.
</details>

<details>
<summary><b>Click to view User Commands</b></summary>

- `/start` - Check if bot is alive.
- `/ping` - Check latency.
- `/help` - Show help menu.

</details>

---

<div align="center">

## ❤️ Donate

</div>

If you find this project useful, consider supporting its development with a donation:

- **TON**: `UQDkCHTN1CA-j_5imVmliDlkqydJhE7nprQZrvFCakr67GEs`
- **USDT TRC20**: `TJWZqPK5haSE8ZdSQeWBPR5uxPSUnS8Hcq`
- **USDT TON**: `UQD8rsWDh3VD9pXVNuEbM_rIAKzV07xDhx-gzdDe0tTWGXan`
- **Telegram Wallet**: [@Ashokshau](https://t.me/Ashokshau)

---

<div align="center">

### 💬 Links

</div>

- 📦 Repo: [TgMusicBot on GitHub](https://github.com/AshokShau/TgMusicBot)
- 💬 Support: [Telegram Group](https://t.me/FallenProjects)
- 🐍 Old version: [TgMusicBot (Python)](https://github.com/AshokShau/TgMusicBot/tree/python)

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/ashokshau">Ashok Shau</a>
</p>
