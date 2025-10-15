#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, chat_cache


@Client.on_message(filters=Filter.command("clear"))
@admins_only(is_bot=True, is_auth=True)
async def clear_queue(c: Client, msg: types.Message) -> None:
    """Handles the /clear command to empty the song queue.

    This command removes all tracks from the current chat's playback queue.
    It requires the user to be an admin or an authorized user.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
    chat_id = msg.chat_id
    if chat_id > 0:
        return None

    if not chat_cache.is_active(chat_id):
        await msg.reply_text("ℹ️ No active playback session found.")
        return None

    if not chat_cache.get_queue(chat_id):
        await msg.reply_text("ℹ️ The queue is already empty.")
        return None

    chat_cache.clear_chat(chat_id)
    reply = await msg.reply_text(f"✅ Queue cleared by {await msg.mention()}")
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply}")
    return None
