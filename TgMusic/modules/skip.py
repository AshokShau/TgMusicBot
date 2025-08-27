#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, call, chat_cache, admins_only
from .utils.play_helpers import del_msg


@Client.on_message(filters=Filter.command("skip"))
@admins_only(is_bot=True, is_auth=True)
async def skip_song(_: Client, msg: types.Message) -> None:
    chat_id = msg.chat_id
    if not chat_cache.is_active(chat_id):
        await msg.reply_text("⏸ No active playback session")
        return None

    await del_msg(msg)
    await call.play_next(chat_id)
    return None
