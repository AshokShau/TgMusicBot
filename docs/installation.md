# 🚀 TgMusicBot Installation Guide

Welcome to the TgMusicBot installation guide! This document provides detailed, step-by-step instructions to help you deploy the bot on your preferred platform.

## Table of Contents
- [Prerequisites](#-prerequisites)
- [Configuration](#-configuration)
- [Deployment Methods](#-deployment-methods)
  - [🐳 Docker (Recommended)](#-docker-recommended)
  - [🔧 Manual Installation](#-manual-installation)
    - [Linux / macOS](#-linux--macos)
    - [Windows](#-windows)

---

## 📋 Prerequisites

Before you begin, ensure you have the following:

- **Telegram API Credentials**:
  - `API_ID` and `API_HASH`: Get these from [my.telegram.org](https://my.telegram.org).
  - `BOT_TOKEN`: Get this from [@BotFather](https://t.me/BotFather) on Telegram.
- **MongoDB URI**: A connection string for your MongoDB database. You can get a free cluster from [MongoDB Atlas](https://www.mongodb.com/cloud/atlas).

---

## ⚙️ Configuration

The bot is configured using a `.env` file. You'll need to create this file and fill it with your credentials.

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/AshokShau/TgMusicBot.git
    cd TgMusicBot
    ```

2.  **Create the `.env` file:**
    ```sh
    cp sample.env .env
    ```

3.  **Edit the `.env` file:**
    Choose one of the following methods to edit the `.env` file and add your credentials.

    - **For beginners (using `nano`):**
      1.  Open the file:
          ```sh
          nano .env
          ```
      2.  Edit the values for `API_ID`, `API_HASH`, `TOKEN`, `MONGO_URI`, etc.
      3.  Save the file by pressing `Ctrl+O`, then `Enter`.
      4.  Exit nano by pressing `Ctrl+X`.

    - **For advanced users (using `vim`):**
      1.  Open the file:
          ```sh
          vi .env
          ```
      2.  Press `i` to enter insert mode.
      3.  Edit the values for `API_ID`, `API_HASH`, `TOKEN`, `MONGO_URI`, etc.
      4.  Press `Esc` to exit insert mode.
      5.  Type `:wq` and press `Enter` to save and quit.

---

## 🚀 Deployment Methods

### 🐳 Docker (Recommended)

Deploying with Docker is the easiest and recommended method.

#### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) installed on your system.

#### Steps
1.  **Clone the repository and create the `.env` file** as described in the [Configuration](#-configuration) section.

2.  **Build the Docker image:**
    ```sh
    docker build -t TgMusicBot .
    ```

3.  **Run the Docker container:**
    ```sh
    docker run -d --name TgMusicBot --env-file .env --restart unless-stopped TgMusicBot
    ```

### 🔧 Manual Installation

#### 🐧 Linux / macOS

##### Prerequisites
- [Go](https://golang.org/doc/install) (version 1.18 or higher)
- [FFmpeg](https://ffmpeg.org/download.html)

##### Steps
1.  **Install prerequisites:**
    - **On Debian/Ubuntu:**
      ```sh
      sudo apt-get update && sudo apt-get install -y golang ffmpeg
      ```
    - **On macOS (using Homebrew):**
      ```sh
      brew install go ffmpeg
      ```

2.  **Clone the repository and create the `.env` file** as described in the [Configuration](#-configuration) section.

3.  **Generate necessary files:**
    ```sh
    go generate
    ```

4.  **Install dependencies and run the bot:**
    ```sh
    go mod tidy
    go run main.go
    ```

#### 🪟 Windows

##### Prerequisites
- [Go](https://golang.org/doc/install) (version 1.18 or higher)
- [FFmpeg](https://ffmpeg.org/download.html)

##### Steps
1.  **Install prerequisites:**
    - Download and install Go from the [official website](https://golang.org/doc/install).
    - Download FFmpeg from the [official website](https://ffmpeg.org/download.html) and add it to your system's PATH.

2.  **Clone the repository** as described in the [Configuration](#-configuration) section.

3.  **Create and edit the `.env` file:**
    - Open Command Prompt or PowerShell.
    - Navigate to the `TgMusicBot` directory.
    - Create the `.env` file:
      ```sh
      copy sample.env .env
      ```
    - Open the `.env` file with Notepad:
      ```sh
      notepad .env
      ```
    - Add your credentials and save the file.

4.  **Generate necessary files:**
    ```sh
    go generate
    ```

5.  **Install dependencies and run the bot:**
    ```sh
    go mod tidy
    go run main.go
    ```
---

That's it! Your TgMusicBot bot should now be running. If you have any questions, feel free to open an issue or join our support group.