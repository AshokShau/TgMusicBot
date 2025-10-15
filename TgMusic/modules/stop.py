#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache


@Client.on_message(filters=Filter.command(["stop", "end"]))
@admins_only(is_bot=True, is_auth=True)
async def stop_song(_: Client, msg: types.Message) -> None:
    """Stop the current playback and clear the queue."""
    chat_id = msg.chat_id
    if not chat_cache.is_active(chat_id):
        await msg.reply_text("⏸ No active playback session")
        return None

    _end = await call.end(chat_id)
    if isinstance(_end, types.Error):
        await msg.reply_text(f"⚠️ <b>Error:</b> {_end.message}")
        return None

    await msg.reply_text(
        f"⏹️ Playback stopped by {await msg.mention()}\n" "🔇 The queue has been cleared"
    )
    return None
