#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from src import db
from src.helpers import get_string, chat_cache, call, MusicServiceWrapper, ChannelPlay
from src.modules.utils import Filter, is_channel_cmd, control_buttons
from src.modules.utils.admins import is_admin

from .play import _get_platform_url, play_music
from .progress_handler import _handle_play_c_data
from .utils.play_helpers import edit_text


@Client.on_updateNewCallbackQuery(filters=Filter.regex(r"(c)?play_\w+"))
async def callback_query(c: Client, message: types.UpdateNewCallbackQuery) -> None:
    """Handle all play control callback queries (skip, stop, pause, resume)."""
    data = message.payload.data.decode()
    user_id = message.sender_user_id
    channel_play = is_channel_cmd(data)
    lang = await db.get_lang(message.chat_id)
    get_msg = await message.getMessage()

    if isinstance(get_msg, types.Error):
        c.logger.warning(get_msg.message)
        return None

    user = await c.getUser(user_id)
    if isinstance(user, types.Error):
        c.logger.warning(user.message)
        return None

    data = data[1:] if channel_play else data
    user_name = user.first_name
    def requires_admin(action: str) -> bool:
        return action in {"play_skip", "play_stop", "play_pause", "play_resume", "play_close"}

    def requires_active_chat(action: str) -> bool:
        return action in {"play_skip", "play_stop", "play_pause", "play_resume", "play_timer"}

    async def send_response(msg: str, alert: bool = False, delete: bool = False, reply_markup=None) -> None:
        if alert:
            await message.answer(msg, show_alert=True)
        else:
            edit_func = message.edit_message_caption if get_msg.caption else message.edit_message_text
            await edit_func(msg, reply_markup=reply_markup)

        if delete:
            _delete = await c.deleteMessages(message.chat_id, [message.message_id], revoke=True)
            if isinstance(_delete, types.Error):
                c.logger.warning("Error deleting message: %s", _delete.message)

    if requires_admin(data) and not await is_admin(message.chat_id, user_id):
        await message.answer(f"⚠️ {get_string('admin_required', lang)}", show_alert=True)
        return None

    chat_id = message.chat_id
    _chat_id = await db.get_channel_id(chat_id) if channel_play else chat_id
    if requires_active_chat(data) and not chat_cache.is_active(_chat_id):
        return await send_response(f"❌ {get_string('no_active_chat', lang)}", alert=True)

    if data == "play_skip":
        result = await call.play_next(_chat_id)
        if isinstance(result, types.Error):
            return await send_response(f"⚠️ {get_string('error_occurred', lang)}\n\n{result.message}", alert=True)
        return await send_response(get_string("song_skipped", lang), delete=True)

    if data == "play_stop":
        result = await call.end(_chat_id)
        if isinstance(result, types.Error):
            return await send_response(result.message, alert=True)
        return await send_response(
            f"<b>➻ {get_string('stream_stopped', lang)}:</b>\n└ {get_string('requested_by', lang)}: {user_name}"
        )

    if data == "play_pause":
        result = await call.pause(_chat_id)
        if isinstance(result, types.Error):
            return await send_response(f"⚠️ {get_string('error_occurred', lang)}\n\n{result.message}", alert=True)
        markup = control_buttons("pause", channel_play) if await db.get_buttons_status(chat_id) else None
        return await send_response(
            f"<b>➻ {get_string('stream_paused', lang)}:</b>\n└ {get_string('requested_by', lang)}: {user_name}",
            reply_markup=markup,
        )

    if data == "play_resume":
        result = await call.resume(_chat_id)
        if isinstance(result, types.Error):
            return await send_response(result.message, alert=True)
        markup = control_buttons("resume", channel_play) if await db.get_buttons_status(chat_id) else None
        return await send_response(
            f"<b>➻ {get_string('stream_resumed', lang)}:</b>\n└ {get_string('requested_by', lang)}: {user_name}",
            reply_markup=markup,
        )

    if data == "play_close":
        _del = await c.deleteMessages(chat_id, [message.message_id], revoke=True)
        if isinstance(_del, types.Error):
            await message.answer(f"Failed to close {_del.message}", show_alert=True)
            return None
        await message.answer(get_string("closed", lang), show_alert=True)
        return None

    if data.startswith("play_c_"):
        return await _handle_play_c_data(data, message, chat_id, user_id, user_name, c)

    # Playing from source
    try:
        _, platform, song_id = data.split("_", 2)
    except ValueError:
        c.logger.error(f"Invalid callback data format: {data}")
        return await send_response(get_string("invalid_request_format", lang), alert=True)

    await message.answer(f"{get_string('playing_song', lang)} {user_name}", show_alert=True)
    reply = await message.edit_message_text(
        f"🎶 {get_string('searching', lang)} ...\n{get_string('requested_by', lang)}: {user_name} 🥀"
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error editing message: {reply}")
        return None

    url = _get_platform_url(platform, song_id)
    if not url:
        c.logger.error(f"Invalid platform: {platform}; data: {data}")
        await edit_text(reply, text=f"⚠️ {get_string('invalid_platform', lang)} {platform}")
        return None

    if song := await MusicServiceWrapper(url).get_info():
        return await play_music(c, reply, song, user_name, channel=ChannelPlay(chat_id=_chat_id, is_channel=channel_play))

    await edit_text(reply, text=get_string("song_not_found", lang))
    return None
