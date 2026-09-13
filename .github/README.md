# TgMusicBot

TgMusicBot is a Telegram music and video bot written in Go. It streams audio and video into Telegram group voice chats using TDLib (`gotdbot`), `gogram`, and `ntgcalls` C bindings.

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
  Built with <code>gotdbot</code>, <code>ntgcalls</code>, and <code>mongo-driver</code>.
</p>

<p align="center">
    <a href="https://heroku.com/deploy?template=https://github.com/AshokShau/TgMusicBot">
        <img src="https://www.herokucdn.com/deploy/button.svg" alt="Heroku Deploy">
    </a>
</p>

---


## Features

- High-performance audio and video streaming into Telegram voice chats.
- Multiple streaming sources including YouTube, Spotify, direct HTTP audio links, and Telegram media files.
- Media playback controls: play, force play, pause, resume, skip, stop, seek, loop, mute, and unmute.
- Queue management and custom user playlists.
- Admin access control and authorization system per chat.
- Autoplay support for endless track playback.
- Docker and Docker Compose deployment support.

## Supported Sources

- YouTube (search queries, track links, playlist links)
- Spotify (track and playlist links via external downloader API)
- Direct audio and video URLs (HTTP/HTTPS streams)
- Telegram audio and video files

## Requirements

- Go 1.26 or higher
- C toolchain (GCC) with CGO enabled (`CGO_ENABLED=1`)
- FFmpeg
- yt-dlp
- Deno
- MongoDB instance (MongoDB Atlas or self-hosted)
- Telegram API ID, API Hash, Bot Token, and Userbot Session String

## Quick Start

1. Clone the repository:

   ```bash
   git clone https://github.com/AshokShau/TgMusicBot.git
   cd TgMusicBot
   ```

2. Copy the sample environment file and configure your credentials:

   ```bash
   cp sample.env .env
   ```

3. Download required TDLib binaries and `ntgcalls` C libraries:

   ```bash
   go run github.com/AshokShau/gotdbot/scripts/tools
   go run setup_ntgcalls.go
   ```

4. Build and start the bot:

   ```bash
   CGO_ENABLED=1 go build -o tgmusic main.go
   ./tgmusic
   ```

For detailed setup instructions on Linux, macOS, Windows, systemd, and Docker, refer to the [Installation Guide](installation.md).

## Configuration Overview

Key environment variables in `.env`:

| Variable              | Required | Description                                                   |
|-----------------------|----------|---------------------------------------------------------------|
| `API_ID`              | Yes      | Telegram API ID from my.telegram.org                          |
| `API_HASH`            | Yes      | Telegram API Hash from my.telegram.org                        |
| `TOKEN`               | Yes      | Telegram Bot Token from @BotFather                            |
| `STRING`              | Yes      | Pyrogram or Telethon session string for the assistant userbot |
| `MONGO_URI`           | Yes      | MongoDB connection string                                     |
| `OWNER_ID`            | Yes      | Telegram user ID of the bot owner                             |
| `LOGGER_ID`           | No       | Telegram chat ID for logging startup and errors               |
| `DEFAULT_SERVICE`     | No       | Default audio provider (`youtube` or `spotify`)               |
| `SONG_DURATION_LIMIT` | No       | Maximum allowed playback duration in seconds (default: 3600)  |
| `ENABLE_VPLAY`        | No       | Toggle video playback support (default: `true`)               |

For a complete list of configuration options, see `sample.env` or the [Installation Guide](installation.md).

## Commands

<details>
<summary>Playback Commands</summary>

| Command                         | Description                                               |
|---------------------------------|-----------------------------------------------------------|
| `/play` or `/p [query/URL]`     | Play audio from YouTube, Spotify, URL, or Telegram file.  |
| `/fplay` or `/fp [query/URL]`   | Force play audio immediately, interrupting current track. |
| `/vplay` or `/v [query/URL]`    | Play video in the voice chat.                             |
| `/fvplay` or `/fvp [query/URL]` | Force play video immediately.                             |
| `/pause`                        | Pause current playback.                                   |
| `/resume`                       | Resume paused playback.                                   |
| `/skip`                         | Skip to the next track in queue.                          |
| `/stop` or `/end`               | Stop playback and clear the queue.                        |
| `/seek [seconds]`               | Seek to a specific timestamp in seconds.                  |
| `/loop [enable/disable]`        | Loop the current track.                                   |
| `/mute`                         | Mute the assistant in the voice chat.                     |
| `/unmute`                       | Unmute the assistant in the voice chat.                   |

</details>

<details>
<summary>Queue & Playlist Commands</summary>

| Command                  | Description                                      |
|--------------------------|--------------------------------------------------|
| `/queue`                 | Display the current playback queue.              |
| `/remove [index]`        | Remove a specific track from the queue.          |
| `/cplist [name]`         | Create a custom playlist.                        |
| `/deleteplaylist [name]` | Delete a custom playlist.                        |
| `/addtoplaylist`         | Add current or replied track to custom playlist. |
| `/removefromplaylist`    | Remove track from custom playlist.               |
| `/playlistinfo [name]`   | View tracks in a custom playlist.                |
| `/myplaylists`           | List all your custom playlists.                  |

</details>

<details>
<summary>Chat & Admin Commands</summary>

| Command            | Description                                          |
|--------------------|------------------------------------------------------|
| `/join` or `/link` | Invite or join the assistant userbot to the chat.    |
| `/auth`            | Grant bot admin rights in chat to a user.            |
| `/removeAuth`      | Revoke bot admin rights from a user.                 |
| `/authList`        | List authorized users in the chat.                   |
| `/settings`        | Open interactive settings menu for chat preferences. |
| `/autoplay`        | Toggle automatic recommendations when queue ends.    |
| `/reload`          | Reload admin cache for the chat.                     |

</details>

<details>
<summary>Owner & Developer Commands</summary>

| Command               | Description                                   |
|-----------------------|-----------------------------------------------|
| `/active_vc` or `/av` | List active voice chats.                      |
| `/broadcast`          | Broadcast message to all served chats.        |
| `/stop_broadcast`     | Cancel ongoing broadcast.                     |
| `/clearass`           | Reset assistant chat assignments.             |
| `/leaveAll`           | Force assistant userbots to leave all chats.  |
| `/logger`             | Toggle logging in log channel.                |
| `/stats`              | View system performance and usage statistics. |
| `/ping`               | Check bot latency and status.                 |

</details>

## Docker Usage

Build and run using Docker Compose:

```bash
docker-compose up -d --build
```

View logs:

```bash
docker-compose logs -f
```

For more Docker options, see the [Installation Guide](installation.md#docker-deployment).

## License

This project is licensed under the GNU General Public License v3.0. See the [LICENSE](../LICENSE) file for details.

## Support & Contact

- Support Group: [Telegram Support](https://t.me/FallenSupport)
- Channel: [Telegram Channel](https://t.me/FallenProjects)
