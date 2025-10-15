#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache

from .utils.play_helpers import del_msg


@Client.on_message(filters=Filter.command("skip"))
@admins_only(is_bot=True, is_auth=True)
async def skip_song(_: Client, msg: types.Message) -> None:
    """Handles the /skip command to skip the current song.

    This command tells the player to stop the current track and immediately
    start playing the next one in the queue.

    Args:
        _ (Client): The pytdbot client instance (unused).
        msg (types.Message): The message object containing the command.
    """
    chat_id = msg.chat_id
    if not chat_cache.is_active(chat_id):
        await msg.reply_text("⏸ No active playback session")
        return None

    await del_msg(msg)
    await call.play_next(chat_id)
    return None
