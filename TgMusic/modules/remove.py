#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, chat_cache, admins_only
from .utils.play_helpers import extract_argument


@Client.on_message(filters=Filter.command("remove"))
@admins_only(is_bot=True, is_auth=True)
async def remove_song(c: Client, msg: types.Message) -> None:
    """Remove a specific track from the playback queue."""
    chat_id = msg.chat_id
    if chat_id > 0:
        return None

    args = extract_argument(msg.text, enforce_digit=True)
    if not chat_cache.is_active(chat_id):
        await msg.reply_text("⏸ No active playback session.")
        return None

    if not args:
        await msg.reply_text(
            "ℹ️ <b>Usage:</b> <code>/remove [track_number]</code>\n"
            "Example: <code>/remove 3</code>"
        )
        return None

    try:
        track_num = int(args)
    except ValueError:
        await msg.reply_text("⚠️ Please enter a valid track number.")
        return None

    _queue = chat_cache.get_queue(chat_id)
    if not _queue:
        await msg.reply_text("📭 The queue is currently empty.")
        return None

    if track_num <= 0 or track_num > len(_queue):
        await msg.reply_text(
            f"⚠️ Invalid track number. Please choose between 1 and {len(_queue)}."
        )
        return None

    chat_cache.remove_track(chat_id, track_num)
    reply = await msg.reply_text(
        f"✅ Track No. {track_num} removed by {await msg.mention()}"
    )

    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply.message}")
    return None
