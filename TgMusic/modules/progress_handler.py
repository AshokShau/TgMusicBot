#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import math
import time

from pytdbot import Client, types

from TgMusic.core import tg, is_admin
from TgMusic.logger import LOGGER

download_progress = {}


def _format_bytes(size: int) -> str:
    """
    Format a size in bytes into a human-readable format.

    Args:
        size: The size in bytes.

    Returns:
        A string containing the size formatted in a human-readable way.
    """
    if size < 1024:
        return f"{size} B"
    for unit in ["KB", "MB", "GB", "TB"]:
        size /= 1024
        if size < 1024:
            return f"{size:.1f} {unit}"
    return f"{size:.1f} PB"


def _format_time(seconds: float) -> str:
    """
    Format a time in seconds into a human-readable format.

    Args:
        seconds: The time in seconds.

    Returns:
        A string containing the time formatted in a human-readable way.
    """
    if seconds < 60:
        return f"{int(seconds)}s"
    minutes, seconds = divmod(seconds, 60)
    if minutes < 60:
        return f"{int(minutes)}m {int(seconds)}s"
    hours, minutes = divmod(minutes, 60)
    return f"{int(hours)}h {int(minutes)}m"


def _create_progress_bar(percentage: int, length: int = 10) -> str:
    """
    Generate a textual progress bar representation.

    Args:
        percentage: The completion percentage of the task.
        length: The total length of the progress bar.

    Returns:
        A string representation of the progress bar, using filled and unfilled
        characters to indicate the progress.
    """
    filled = round(length * percentage / 100)
    return "⬢" * filled + "⬡" * (length - filled)


def _calculate_update_interval(file_size: int, speed: float) -> float:
    """
    Calculates the interval between progress updates in seconds.

    The interval is determined by the file size and download speed. For smaller
    files (less than 5MB), the interval is fixed at 1 second. For larger files,
    the interval is calculated based on the file size and speed, with a minimum
    of 1 second and a maximum of 5 seconds.

    Args:
        file_size: The size of the file in bytes.
        speed: The download speed in bytes per second.

    Returns:
        The interval between progress updates in seconds.
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
    """
    Generates the "Stop Downloading" inline button for a specific unique ID.

    Args:
        unique_id: The unique ID of the download.

    Returns:
        A ReplyMarkupInlineKeyboard with the "Stop Downloading" button.
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
    """
    Checks if a progress update should be sent.

    Args:
        progress: A dictionary containing the current progress information.
        now: The current time in seconds.
        completed: Whether the task has completed.

    Returns:
        True if an update should be sent, False otherwise.
    """
    return now >= progress["next_update"] or completed


def _build_progress_text(
    filename: str, total: int, downloaded: int, speed: float
) -> str:
    """
    Build a progress update message for a download task.

    This function generates a formatted string indicating the current progress
    of a download task. It displays the filename, total size, current progress,
    speed, and estimated time of arrival (ETA) of the download.

    Args:
        filename: The name of the downloaded file.
        total: The total size of the file in bytes.
        downloaded: The amount of data downloaded so far in bytes.
        speed: The current download speed in bytes per second.

    Returns:
        A string containing the formatted progress update message.
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
    """
    Build a completion message for a download task.

    This function generates a formatted string indicating the completion
    of a download task. It displays the filename, total size, time taken,
    and average speed of the download.

    Args:
        filename: The name of the downloaded file.
        total: The total size of the file in bytes.
        duration: The time taken to complete the download in seconds.

    Returns:
        A string containing the formatted completion message.
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
    """
    Handles file download progress updates.

    This function is called when the Telegram Client receives a file download
    progress update. It extracts the necessary information from the update,
    calculates the download speed and ETA, and sends a progress update message
    to the user. If the download is complete, it sends a final "download complete"
    message and removes the download from the progress tracker.

    Args:
        client: The Telegram Client instance.
        update: The file download progress update.

    Returns:
        None
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
    """
    Handle play control callback data for cancelling a file download.

    This function checks if the user is an admin, and if so, attempts to
    cancel an ongoing file download based on the provided callback data.
    It retrieves the metadata associated with the file and cancels the
    download if it is still in progress. The user is notified of the
    success or failure of the cancellation.

    Args:
        data: The callback data containing the file ID.
        message: The message object to send responses to the user.
        chat_id: The ID of the chat where the command was issued.
        user_id: The ID of the user who issued the command.
        user_name: The name of the user who issued the command.
        c: The Telegram Client (pytdbot.Client) instance.

    Returns:
        None
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
