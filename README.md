# 🎵 TgMusicBot – Telegram Music Bot [![Stars](https://img.shields.io/github/stars/AshokShau/TgMusicBot?style=social)](https://github.com/AshokShau/TgMusicBot/stargazers)

**TgMusicBot** is a high-performance Telegram music bot designed for seamless music streaming in voice chats. It leverages a powerful combination of libraries, using `pytdbot` for efficient interaction with the Telegram Bot API and a multi-assistant architecture powered by `pyrogram` and `py-tgcalls` for robust, low-latency audio and video playback.

It supports a wide range of music sources, including YouTube, Spotify, JioSaavn, Apple Music, and SoundCloud, making it a versatile solution for any Telegram community.

<p align="center">
  <!-- GitHub Stars -->
  <a href="https://github.com/AshokShau/TgMusicBot/stargazers">
    <img src="https://img.shields.io/github/stars/AshokShau/TgMusicBot?style=for-the-badge&color=black&logo=github" alt="Stars"/>
  </a>
  
  <!-- GitHub Forks -->
  <a href="https://github.com/AshokShau/TgMusicBot/network/members">
    <img src="https://img.shields.io/github/forks/AshokShau/TgMusicBot?style=for-the-badge&color=black&logo=github" alt="Forks"/>
  </a>

  <!-- Last Commit -->
  <a href="https://github.com/AshokShau/TgMusicBot/commits/AshokShau">
    <img src="https://img.shields.io/github/last-commit/AshokShau/TgMusicBot?style=for-the-badge&color=blue" alt="Last Commit"/>
  </a>

  <!-- Repo Size -->
  <a href="https://github.com/AshokShau/TgMusicBot">
    <img src="https://img.shields.io/github/repo-size/AshokShau/TgMusicBot?style=for-the-badge&color=success" alt="Repo Size"/>
  </a>

  <!-- Language -->
  <a href="https://www.python.org/">
    <img src="https://img.shields.io/badge/Written%20in-Python-orange?style=for-the-badge&logo=python" alt="Python"/>
  </a>

  <!-- License -->
  <a href="https://github.com/AshokShau/TgMusicBot/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/AshokShau/TgMusicBot?style=for-the-badge&color=blue" alt="License"/>
  </a>

  <!-- Open Issues -->
  <a href="https://github.com/AshokShau/TgMusicBot/issues">
    <img src="https://img.shields.io/github/issues/AshokShau/TgMusicBot?style=for-the-badge&color=red" alt="Issues"/>
  </a>

  <!-- Pull Requests -->
  <a href="https://github.com/AshokShau/TgMusicBot/pulls">
    <img src="https://img.shields.io/github/issues-pr/AshokShau/TgMusicBot?style=for-the-badge&color=purple" alt="PRs"/>
  </a>

  <!-- GitHub Workflow CI -->
  <a href="https://github.com/AshokShau/TgMusicBot/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/AshokShau/TgMusicBot/code-fixer.yml?style=for-the-badge&label=CI&logo=github" alt="CI Status"/>
  </a>
</p>

<p align="center">
   <img src="https://raw.githubusercontent.com/AshokShau/TgMusicBot/master/.github/images/thumb.png" alt="thumbnail" width="320" height="320">
</p>

### 🔥 Live Bot: [@FallenBeatzBot](https://t.me/FallenBeatzBot)

---

## ✨ Key Features

| Feature                       | Description                                         |
|-------------------------------|-----------------------------------------------------|
| **🎧 Multi-Platform Support** | YouTube, Spotify, Apple Music, SoundCloud, JioSaavn |
| **📜 Playlist Management**    | Queue system with auto-play                         |
| **🎛️ Advanced Controls**     | Volume, loop, seek, skip, pause/resume              |
| **🌐 Multi-Language**         | English, Hindi, Spanish, Arabic support             |
| **⚡ Low Latency**             | Optimized with PyTgCalls                            |
| **🐳 Docker Ready**           | One-click deployment                                |
| **🔒 Anti-Ban**               | Cookie & API-based authentication                   |

---

## 🏛️ Project Structure

A brief overview of the key directories in this project:

```
TgMusicBot/
├── TgMusic/
│   ├── core/         # Core logic: call handling, database, API clients, config
│   ├── modules/      # Bot commands and feature modules
│   │   ├── utils/    # Utility functions for modules
│   │   └── ...
│   ├── __init__.py   # Main bot class and initialization
│   ├── __main__.py   # Entry point for running the bot
│   └── logger.py     # Logging configuration
├── .env            # Environment variables (create from sample.env)
├── Dockerfile        # For building the Docker image
├── README.md         # This file
└── ...
```
- **`TgMusic/core`**: Contains the essential backend components. This is where the main logic for handling calls, database interactions, and communication with external music services resides.
- **`TgMusic/modules`**: Holds the individual command handlers. Each `.py` file typically corresponds to a specific bot command (e.g., `play.py`, `skip.py`) or a feature set (e.g., `auth.py`).

---

## 🚀 Quick Deploy

[![Deploy on Heroku](https://img.shields.io/badge/Deploy%20on%20Heroku-430098?style=for-the-badge&logo=heroku)](https://heroku.com/deploy?template=https://github.com/AshokShau/TgMusicBot)

---

## 📦 Installation Methods


<details>

<summary><strong>📌 Docker Installation (Recommended) (Click to expand)</strong></summary>

### 🐳 Prerequisites
1. Install Docker:
   - [Linux](https://docs.docker.com/engine/install/)
   - [Windows/Mac](https://docs.docker.com/desktop/install/)

### 🚀 Quick Setup
1. Clone the repository:
   ```sh
   git clone https://github.com/AshokShau/TgMusicBot.git && cd TgMusicBot
   ```

### 🔧 Configuration
1. Prepare environment file:
   ```sh
   cp sample.env .env
   ```

2. Edit configuration (choose one method):
   - **Beginner-friendly (nano)**:
     ```sh
     nano .env
     ```
     - Edit values
     - Save: `Ctrl+O` → Enter → `Ctrl+X`

   - **Advanced (vim)**:
     ```sh
     vi .env
     ```
     - Press `i` to edit
     - Save: `Esc` → `:wq` → Enter

### 🏗️ Build & Run
1. Build Docker image:
   ```sh
   docker build -t tgmusicbot .
   ```

2. Run container (auto-restarts on crash/reboot):
   ```sh
   docker run -d --name tgmusicbot --env-file .env --restart unless-stopped tgmusicbot
   ```

### 🔍 Monitoring
1. Check logs:
   ```sh
   docker logs -f tgmusicbot
   ```
   (Exit with `Ctrl+C`)

### ⚙️ Management Commands
- **Stop container**:
  ```sh
  docker stop tgmusicbot
  ```

- **Start container**:
  ```sh
  docker start tgmusicbot
  ```

- **Update the bot**:
  ```sh
  docker stop tgmusicbot
  docker rm tgmusicbot
  git pull origin master
  docker build -t tgmusicbot .
  docker run -d --name tgmusicbot --env-file .env --restart unless-stopped tgmusicbot
  ```

</details>


<details>
<summary><strong>📌 Step-by-Step Installation Guide (Click to Expand)</strong></summary>

### 🛠️ System Preparation
1. **Update your system** (Recommended):
   ```sh
   sudo apt-get update && sudo apt-get upgrade -y
   ```

2. **Install essential tools**:
   ```sh
   sudo apt-get install git python3-pip ffmpeg tmux -y
   ```

### ⚡ Quick Setup
1. **Install UV package manager**:
   ```sh
   pip3 install uv
   ```

2. **Clone the repository**:
   ```sh
   git clone https://github.com/AshokShau/TgMusicBot.git && cd TgMusicBot
   ```

### 🐍 Python Environment
1. **Create virtual environment**:
   ```sh
   uv venv
   ```

2. **Activate environment**:
   - Linux/Mac: `source .venv/bin/activate`
   - Windows (PowerShell): `.\.venv\Scripts\activate`

3. **Install dependencies**:
   ```sh
   uv pip install -e .
   ```

### 🔐 Configuration
1. **Setup environment file**:
   ```sh
   cp sample.env .env
   ```

2. **Edit configuration** (Choose one method):
   - **For beginners** (nano editor):
     ```sh
     nano .env
     ```
     - Edit values
     - Save: `Ctrl+O` → Enter → `Ctrl+X`

   - **For advanced users** (vim):
     ```sh
     vi .env
     ```
     - Press `i` to edit
     - Save: `Esc` → `:wq` → Enter

### 🤖 Running the Bot
1. **Start in tmux session** (keeps running after logout):
   ```sh
   tmux new -s musicbot
   start
   ```

   **Tmux Cheatsheet**:
   - Detach: `Ctrl+B` then `D`
   - Reattach: `tmux attach -t musicbot`
   - Kill session: `tmux kill-session -t musicbot`

### 🔄 After Updates
To restart the bot:
```sh
tmux attach -t musicbot
# Kill with Ctrl+C
start
```

</details>

---

## ⚙️ Configuration Guide

<details>
<summary><b>🔑 Required Variables (Click to expand)</b></summary>

| Variable     | Description                         | How to Get                                                               |
|--------------|-------------------------------------|--------------------------------------------------------------------------|
| `API_ID`     | Telegram App ID                     | [my.telegram.org](https://my.telegram.org/apps)                          |
| `API_HASH`   | Telegram App Hash                   | [my.telegram.org](https://my.telegram.org/apps)                          |
| `TOKEN`      | Bot Token                           | [@BotFather](https://t.me/BotFather)                                     |
| `STRING1-10` | Pyrogram Sessions (Only 1 Required) | [@StringFatherBot](https://t.me/StringFatherBot)                         |
| `MONGO_URI`  | MongoDB Connection                  | [MongoDB Atlas](https://cloud.mongodb.com)                               |
| `OWNER_ID`   | User ID of the bot owner            | [@GuardxRobot](https://t.me/GuardxRobot) and type `/id`                  |
| `LOGGER_ID`  | Group ID of the bot logger          | Add [@GuardxRobot](https://t.me/GuardxRobot) to the group and type `/id` |

</details>

<details>
<summary><b>🔧 Optional Variables (Click to expand)</b></summary>

| Variable           | Description                                                       | How to Get                                                                                                                                                              |
|--------------------|-------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `API_URL`          | API URL                                                           | Start [@FallenApiBot](https://t.me/FallenApiBot)                                                                                                                        |
| `API_KEY`          | API Key                                                           | Start [@FallenApiBot](https://t.me/FallenApiBot) and type `/apikey`                                                                                                     |
| `MIN_MEMBER_COUNT` | Minimum number of members required to use the bot                 | Default: 50                                                                                                                                                             |
| `PROXY`            | Proxy URL for the bot if you want to use it for yt-dlp (Optional) | Any online service                                                                                                                                                      |
| `COOKIES_URL`      | Cookies URL for the bot                                           | [![Cookie Guide](https://img.shields.io/badge/Guide-Read%20Here-blue?style=flat-square)](https://github.com/AshokShau/TgMusicBot/blob/master/TgMusic/cookies/README.md) |
| `DEFAULT_SERVICE`  | Default search platform (Options: youtube, spotify, jiosaavn)     | Default: youtube                                                                                                                                                        |
| `SUPPORT_GROUP`    | Telegram Group Link                                               | Default: https://t.me/GuardxSupport                                                                                                                                     |
| `SUPPORT_CHANNEL`  | Telegram Channel Link                                             | Default: https://t.me/FallenProjects                                                                                                                                    |
| `AUTO_LEAVE`       | Leave all chats for all userbot clients                           | Default: False                                                                                                                                                          |
| `NO_UPDATES`       | Disable updates                                                   | Default: False                                                                                                                                                          |
| `START_IMG`        | Start Image URL                                                   | Default: [IMG](https://i.pinimg.com/1200x/e8/89/d3/e889d394e0afddfb0eb1df0ab663df95.jpg)                                                                                |
| `DEVS`             | List of Developer User IDs (space-separated)                      | [@GuardxRobot](https://t.me/GuardxRobot) and type `/id`. Example: `5938660179 5956803759`                                                                                |

</details>

---

## 🤖 Bot Commands

### ▶️ Playback Commands
| Command | Description |
|---|---|
| `/play [song/url]` | Plays a song from YouTube, Spotify, etc., or by search term. |
| `/vplay [video/url]`| Plays a video in the voice chat. |
| `/skip` | Skips the current track and plays the next in queue. |
| `/pause` | Pauses the current playback. |
| `/resume` | Resumes the paused playback. |
| `/stop` or `/end` | Stops playback, clears the queue, and leaves the voice chat. |

### 📋 Queue Management
| Command | Description |
|---|---|
| `/queue` | Shows the current list of tracks in the queue. |
| `/loop [1-10]` | Sets the current song to repeat a number of times. Use `/loop 0` to disable. |
| `/clear` | Empties the entire playback queue. |
| `/remove [number]` | Removes a specific track from the queue by its number. |

### ⚙️ Playback Settings
| Command | Description |
|---|---|
| `/volume [1-200]` | Adjusts the playback volume. |
| `/speed [0.5-4.0]`| Changes the playback speed. |
| `/seek [seconds]` | Seeks forward in the track by a number of seconds. |
| `/mute` | Mutes the bot in the voice chat. |
| `/unmute` | Unmutes the bot in the voice chat. |

### 🔐 Permissions (Admin)
| Command | Description |
|---|---|
| `/auth [reply]` | Authorizes a user to use admin commands. |
| `/unauth [reply]`| Revokes a user's authorization. |
| `/authlist` | Lists all authorized users in the chat. |

### 👑 Chat Owner Tools
| Command | Description |
|---|---|
| `/buttons [on/off]` | Toggles the visibility of player control buttons. |
| `/thumb [on/off]` | Toggles the generation of "Now Playing" thumbnails. |
| `/playtype [0/1]` | Sets the play mode (0 for direct play, 1 for selection menu). |

---

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

**Note:** Minor typo fixes will be closed. Focus on meaningful contributions.

---

## 📜 License

AGPL-3.0 © [AshokShau](https://github.com/AshokShau).  
[![License](https://img.shields.io/github/license/AshokShau/TgMusicBot?color=blue)](LICENSE)

---

## 💖 Support

Help keep this project alive!  
[![Telegram](https://img.shields.io/badge/Chat-Support%20Group-blue?logo=telegram)](https://t.me/GuardxSupport)  
[![Donate](https://img.shields.io/badge/Donate-Crypto/PayPal-ff69b4)](https://t.me/AshokShau)

---

## 🔗 Connect

[![GitHub](https://img.shields.io/badge/Follow-GitHub-black?logo=github)](https://github.com/AshokShau)  
[![Channel](https://img.shields.io/badge/Updates-Channel-blue?logo=telegram)](https://t.me/FallenProjects)

---