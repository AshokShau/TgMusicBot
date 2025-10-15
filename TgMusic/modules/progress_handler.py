#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import math
import time

from pytdbot import Client, types

from TgMusic.core import is_admin, tg
from TgMusic.logger import LOGGER

download_progress = {}


def _format_bytes(size: int) -> str:
    """Formats a size in bytes into a human-readable string (KB, MB, GB, etc.).

    Args:
        size (int): The size in bytes.

    Returns:
        str: A human-readable string representing the size.
    """
    if size < 1024:
        return f"{size} B"
    for unit in ["KB", "MB", "GB", "TB"]:
        size /= 1024
        if size < 1024:
            return f"{size:.1f} {unit}"
    return f"{size:.1f} PB"


def _format_time(seconds: float) -> str:
    """Formats a duration in seconds into a human-readable string (e.g., 2m 30s).

    Args:
        seconds (float): The duration in seconds.

    Returns:
        str: A human-readable string representing the time.
    """
    if seconds < 60:
        return f"{int(seconds)}s"
    minutes, seconds = divmod(seconds, 60)
    if minutes < 60:
        return f"{int(minutes)}m {int(seconds)}s"
    hours, minutes = divmod(minutes, 60)
    return f"{int(hours)}h {int(minutes)}m"


def _create_progress_bar(percentage: int, length: int = 10) -> str:
    """Generates a textual progress bar.

    Args:
        percentage (int): The completion percentage (0-100).
        length (int): The total character length of the bar. Defaults to 10.

    Returns:
        str: A string representing the progress bar (e.g., "⬢⬢⬢⬡⬡⬡⬡⬡⬡⬡").
    """
    filled = round(length * percentage / 100)
    return "⬢" * filled + "⬡" * (length - filled)


def _calculate_update_interval(file_size: int, speed: float) -> float:
    """Calculates a dynamic interval for sending progress updates.

    The goal is to avoid spamming with updates for very fast downloads or
    small files, while still providing timely feedback for large or slow
    downloads. The interval is adjusted based on file size and download speed.

    Args:
        file_size (int): The total size of the file in bytes.
        speed (float): The current download speed in bytes per second.

    Returns:
        float: The recommended interval in seconds for the next update.
    """
    if file_size < 5 * 1024 * 1024:
        base = 1.0
    else:
        scale = min(math.log10(file_size / (5 * 1024 * 1024)), 2)
        base = 1.0 + scale * 2.0

    speed_mod = (
        max(0.5, 2.0 - (speed / (5 * 1024 * 1024))) if speed > 1024 * 1024 else 1.0
    )
    return min(max(base * speed_mod, 1.0), 5.0)


def _get_button(unique_id: str) -> types.ReplyMarkupInlineKeyboard:
    """Creates an inline keyboard with a "Stop Downloading" button.

    Args:
        unique_id (str): The unique ID of the download to be included in the
            callback data for the button.

    Returns:
        types.ReplyMarkupInlineKeyboard: The generated inline keyboard.
    """
    return types.ReplyMarkupInlineKeyboard(
        [
            [
                types.InlineKeyboardButton(
                    text="✗ Stop Downloading",
                    type=types.InlineKeyboardButtonTypeCallback(
                        f"play_c_{unique_id}".encode()
                    ),
                )
            ]
        ]
    )


def _should_update(progress: dict, now: float, completed: bool) -> bool:
    """Determines if it's time to send a progress update message.

    Args:
        progress (dict): A dictionary containing the progress state, including
            the `next_update` timestamp.
        now (float): The current time.
        completed (bool): A flag indicating if the download is complete.

    Returns:
        bool: True if an update should be sent, False otherwise.
    """
    return now >= progress["next_update"] or completed


def _build_progress_text(
    filename: str, total: int, downloaded: int, speed: float
) -> str:
    """Constructs the text for a download progress message.

    Args:
        filename (str): The name of the file being downloaded.
        total (int): The total size of the file in bytes.
        downloaded (int): The number of bytes downloaded so far.
        speed (float): The current download speed in bytes/sec.

    Returns:
        str: The formatted progress message.
    """
    percentage = min(100, int((downloaded / total) * 100))
    eta = int((total - downloaded) / speed) if speed > 0 else -1
    return (
        f"📥 <b>Downloading:</b> <code>{filename}</code>\n"
        f"💾 <b>Size:</b> {_format_bytes(total)}\n"
        f"📊 <b>Progress:</b> {percentage}% {_create_progress_bar(percentage)}\n"
        f"🚀 <b>Speed:</b> {_format_bytes(int(speed))}/s\n"
        f"⏳ <b>ETA:</b> {_format_time(eta) if eta >= 0 else 'Calculating...'}"
    )


def _build_complete_text(filename: str, total: int, duration: float) -> str:
    """Constructs the text for a download completion message.

    Args:
        filename (str): The name of the downloaded file.
        total (int): The total size of the file in bytes.
        duration (float): The total time taken for the download in seconds.

    Returns:
        str: The formatted completion message.
    """
    avg_speed = total / max(duration, 1e-6)
    return (
        f"✅ <b>Download Complete:</b> <code>{filename}</code>\n"
        f"💾 <b>Size:</b> {_format_bytes(total)}\n"
        f"⏱ <b>Time Taken:</b> {_format_time(duration)}\n"
        f"⚡ <b>Average Speed:</b> {_format_bytes(int(avg_speed))}/s"
    )


@Client.on_updateFile()
async def update_file(client: Client, update: types.UpdateFile):
    """Handles `updateFile` events from `pytdbot` to show download progress.

    This function is the core of the download progress reporting system. It
    is triggered by the tdlib client whenever a chunk of a file is downloaded.
    It calculates progress, speed, and ETA, and edits the original message
    to show the live status to the user.

    Args:
        client (Client): The pytdbot client instance.
        update (types.UpdateFile): The file update object from pytdbot.
    """
    file = update.file
    unique_id = file.remote.unique_id
    meta = tg.get_cached_metadata(unique_id)
    if not meta:
        return

    chat_id = meta["chat_id"]
    filename = meta["filename"]
    message_id = meta["message_id"]
    file_id = file.id
    now = time.time()

    total = file.size or 1
    downloaded = file.local.downloaded_size

    if file_id not in download_progress:
        download_progress[file_id] = {
            "start_time": now,
            "last_update": now,
            "last_size": downloaded,
            "next_update": now + 1.0,
            "last_speed": 0,
        }

    progress = download_progress[file_id]

    if not _should_update(progress, now, file.local.is_downloading_completed):
        return

    elapsed = now - progress["last_update"]
    delta = downloaded - progress["last_size"]
    speed = delta / elapsed if elapsed > 0 else 0
    interval = _calculate_update_interval(total, speed)

    progress.update(
        {
            "next_update": now + interval,
            "last_update": now,
            "last_size": downloaded,
            "last_speed": speed,
        }
    )

    button_markup = _get_button(unique_id)

    if not file.local.is_downloading_completed:
        progress_text = _build_progress_text(filename, total, downloaded, speed)
        parsed = await client.parseTextEntities(
            progress_text, types.TextParseModeHTML()
        )
        edit = await client.editMessageText(
            chat_id, message_id, button_markup, types.InputMessageText(parsed)
        )
        if isinstance(edit, types.Error):
            LOGGER.error("Progress update error: %s", edit)
        return

    # Completed download
    duration = now - progress["start_time"]
    complete_text = _build_complete_text(filename, total, duration)
    parsed = await client.parseTextEntities(complete_text, types.TextParseModeHTML())
    done = await client.editMessageText(
        chat_id, message_id, button_markup, types.InputMessageText(parsed)
    )
    if isinstance(done, types.Error):
        LOGGER.error("Download complete update error: %s", done)

    download_progress.pop(file_id, None)


async def _handle_play_c_data(
    data: str,
    message: types.UpdateNewCallbackQuery,
    chat_id: int,
    user_id: int,
    user_name: str,
    c: Client,
):
    """Handles the callback query for cancelling a file download.

    This function is triggered when a user clicks the "Stop Downloading"
    button. It verifies that the user is an admin and then cancels the
    corresponding download task.

    Args:
        data (str): The callback data from the button, containing the unique
            file ID to cancel.
        message (types.UpdateNewCallbackQuery): The callback query object.
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who clicked the button.
        user_name (str): The first name of the user.
        c (Client): The pytdbot client instance.
    """
    if not await is_admin(chat_id, user_id):
        await message.answer(
            "⚠️ You must be an admin to use this command.", show_alert=True
        )
        return

    _, _, file_id = data.split("_", 2)
    meta = tg.get_cached_metadata(file_id)
    if not meta:
        await message.answer(
            "Looks like this file already downloaded.", show_alert=True
        )
        return

    file_info = await c.getRemoteFile(meta["remote_file_id"])
    if isinstance(file_info, types.Error):
        await message.answer("Failed to get file info", show_alert=True)
        LOGGER.error("Failed to get file info: %s", file_info.message)
        return

    ok = await c.cancelDownloadFile(file_info.id)
    if isinstance(ok, types.Error):
        await message.answer(
            f"Failed to cancel download. {ok.message}", show_alert=True
        )
        return

    await message.answer("Download cancelled.", show_alert=True)
    await message.edit_message_text(
        f"Download cancelled.\nRequested by: {user_name} 🥀"
    )
