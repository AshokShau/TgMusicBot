#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

__all__ = [
    "Filter",
    "sec_to_min",
    "get_audio_duration",
    "SupportButton",
    "send_logger",
    "user_status_cache",
    "chat_invite_cache",
    "join_ub",
    "check_user_status",
    "ChatMemberStatus",
    "is_channel_cmd",
    "control_buttons",
]

import asyncio
import json

from pytdbot import Client, types

from ._filters import Filter
from ._join_ub import user_status_cache, ChatMemberStatus, join_ub, check_user_status, chat_invite_cache
from .buttons import SupportButton, control_buttons
from ... import config
from ...helpers import CachedTrack
from ...logger import LOGGER


def is_channel_cmd(text: str) -> bool:
    if not text:
        return False
    text = text.strip().lower()
    return any(text.startswith(prefix) for prefix in ("/c", "!c", ".c", "cplay"))


def sec_to_min(seconds):
    """
    Convert seconds to minutes:second format.
    """
    try:
        minutes = int(seconds // 60)
        remaining_seconds = int(seconds % 60)
        return f"{minutes}:{remaining_seconds:02}"
    except Exception as e:
        LOGGER.warning("Failed to convert seconds to minutes:seconds format: %s", e)
        return None


async def send_logger(client: Client, chat_id, song: CachedTrack):
    """
    Send a message to the logger channel when a song is played.

    Args:
        client (Client): The client to send the message with.
        chat_id (int): The ID of the chat that the song is being played in.
        song (CachedTrack): The song that is being played.

    Returns:
        None
    """
    if not chat_id or not song or chat_id == config.LOGGER_ID or config.LOGGER_ID == 0:
        return

    text = (
        f"<b>Song Playing</b> in <code>{chat_id}</code>\n\n"
        f"▶️ <b>Now Playing:</b> <a href='{song.url}'>{song.name}</a>\n\n"
        f"• <b>Duration:</b> {sec_to_min(song.duration)}\n"
        f"• <b>Requested by:</b> {song.user}\n"
        f"• <b>Platform:</b> {song.platform}"
    )

    msg = await client.sendTextMessage(
        config.LOGGER_ID, text, disable_web_page_preview=True, disable_notification=True
    )
    if isinstance(msg, types.Error):
        LOGGER.error("Error sending message: %s", msg)
    return


async def get_audio_duration(file_path):
    try:
        proc = await asyncio.create_subprocess_exec(
            "ffprobe",
            "-v",
            "quiet",
            "-print_format",
            "json",
            "-show_format",
            "-show_streams",
            file_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await proc.communicate()
        data = json.loads(stdout)
        duration = float(data["format"]["duration"])
        return int(duration)
    except Exception as e:
        LOGGER.warning("Failed to get audio duration using ffprobe: %s", e)
        return 0
