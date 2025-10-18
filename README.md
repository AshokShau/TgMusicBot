<div align="center">

# 🎵 TgMusicBot — Telegram Music Bot

**A high-performance, open-source Telegram Music Bot written in Go — stream music and video in Telegram voice chats effortlessly.**

<p>
  <a href="https://github.com/AshokShau/TgMusicBot/stargazers">
    <img src="https://img.shields.io/github/stars/AshokShau/TgMusicBot?style=for-the-badge&color=ffd700&logo=github" alt="Stars">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/network/members">
    <img src="https://img.shields.io/github/forks/AshokShau/TgMusicBot?style=for-the-badge&color=8a2be2&logo=github" alt="Forks">
  </a>
  <a href="https://github.com/AshokShau/TgMusicBot/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/AshokShau/TgMusicBot?style=for-the-badge&color=4169e1" alt="License">
  </a>
  <a href="https://goreportcard.com/report/github.com/AshokShau/TgMusicBot">
    <img src="https://goreportcard.com/badge/github.com/AshokShau/TgMusicBot?style=for-the-badge" alt="Go Report Card">
  </a>
  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/Written%20in-Go-00ADD8?style=for-the-badge&logo=go" alt="Go">
  </a>
</p>

TgMusicBot leverages a powerful combination of Go libraries — using `gogram` for efficient Telegram Bot API integration and `ntgcalls` for robust, low-latency audio and video playback.  
It supports streaming from popular sources like YouTube, making it a complete solution for Telegram music lovers and communities.

</div>

---

<div align="center">

## ✨ Key Features

| Feature                       | Description                                                             |
|-------------------------------|-------------------------------------------------------------------------|
| **🎧 Multi-Platform Support** | Stream directly from YouTube, Spotify, Apple Music, SoundCloud and more |
| **📜 Playlist Management**    | Queue system with auto-play & next-track handling                       |
| **🎛️ Advanced Controls**     | Volume, loop, seek, skip, pause/resume                                  |
| **⚡ Low Latency**             | Optimized audio with `ntgcalls`                                         |
| **🐳 Docker Ready**           | Deploy anywhere in one click                                            |
| **🧠 Built with Go**          | Stable, concurrent, and memory-efficient                                |

</div>

---

## 🚀 Getting Started

### 🔧 Manual Setup

For manual setup instructions for Linux, macOS, and Windows, please see the **[Installation Guide](docs/installation.md)**.

The guide provides comprehensive instructions for deploying the bot using:
- **🐳 Docker (Recommended)**
- **🔧 Manual Installation (Linux, macOS, and Windows)**

---

<div align="center">

## ⚙️ Configuration

</div>

Copy `.env.sample` → `.env` and fill the required values:

| Variable          | Description                  | How to Get                                      |
|-------------------|------------------------------|-------------------------------------------------|
| `API_ID`          | Your Telegram app’s API ID   | [my.telegram.org](https://my.telegram.org/apps) |
| `API_HASH`        | Your Telegram app’s API hash | [my.telegram.org](https://my.telegram.org/apps) |
| `TOKEN`           | Your bot token               | [@BotFather](https://t.me/BotFather)            |
| `SESSION_STRINGS` | Your user session string     | Use a gogram session generator                  |
| `MONGO_URI`       | MongoDB connection string    | [MongoDB Atlas](https://cloud.mongodb.com)      |
| `OWNER_ID`        | Your Telegram user ID        | [@userinfobot](https://t.me/userinfobot)        |
| `LOGGER_ID`       | Group chat ID for logs       | Add bot to group & check `chat_id`              |

---

<div align="center">

## 🤖 Commands

</div>

| Command              | Description                         |
|----------------------|-------------------------------------|
| `/play [song/url]`   | Play audio from YouTube or a URL    |
| `/vplay [video/url]` | Play video in the voice chat        |
| `/skip`              | Skip the current track              |
| `/pause`             | Pause playback                      |
| `/resume`            | Resume playback                     |
| `/stop` or `/end`    | Stop and clear queue                |
| `/queue`             | Show the active queue               |
| `/loop [on/off]`     | Loop the current track              |
| `/auth [reply]`      | Authorize a user for admin commands |
| `/unauth [reply]`    | Remove user authorization           |
| `/authlist`          | List authorized users               |

---

<div align="center">

## 🧩 Project Structure

</div>

```
TgMusicBot/
├── pkg/
│   ├── config/       # Configuration loading
│   ├── core/         # Core logic: database, caching, etc.
│   ├── handlers/     # Telegram command handlers
│   └── vc/           # Voice chat management
├── sample.env        # Example environment config
├── Dockerfile        # Docker build configuration
├── go.mod            # Go module definition
└── main.go           # Application entry point
```

---

<div align="center">

## 🤝 Contributing

</div>

Contributions are **welcome**!  
To contribute:

1. **Fork** the repo  
2. **Create** your feature branch → `git checkout -b feature/AmazingFeature`  
3. **Commit** changes → `git commit -m 'Add some AmazingFeature'`  
4. **Push** → `git push origin feature/AmazingFeature`  
5. **Open a pull request**

⭐ If you like this project, please **star** it — it helps others find it!

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

## 📜 License

</div>

Licensed under the **GNU General Public License (GPL v3)**.  
See the [LICENSE](LICENSE) file for details.

---

<div align="center">

### 💬 Links

</div>

- 📦 Repo: [TgMusicBot on GitHub](https://github.com/AshokShau/TgMusicBot)
- 💬 Support: [Telegram Group](https://t.me/FallenProjects)
- 🐍 Old version: [TgMusicBot (Python)](https://github.com/AshokShau/TgMusicBot/tree/python)
