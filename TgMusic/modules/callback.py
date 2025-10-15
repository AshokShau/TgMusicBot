#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache, control_buttons, db

from ..core import DownloaderWrapper
from .play import _get_platform_url, play_music
from .progress_handler import _handle_play_c_data
from .utils.play_helpers import edit_text


@Client.on_updateNewCallbackQuery(filters=Filter.regex(r"vcplay_\w+"))
async def callback_query_vc_play(
    c: Client, message: types.UpdateNewCallbackQuery
) -> None:
    """Handles callback queries related to starting playback from search results.

    This function is triggered when a user clicks a button from the inline
    search results. It parses the callback data to identify the song and
    platform, then initiates the playback process.

    Args:
        c (Client): The pytdbot client instance.
        message (types.UpdateNewCallbackQuery): The callback query update.
    """
    data = message.payload.data.decode()
    chat_id = message.chat_id

    if data == "vcplay_close":
        delete_result = await c.deleteMessages(
            chat_id, [message.message_id], revoke=True
        )
        if isinstance(delete_result, types.Error):
            await message.answer(
                f"⚠️ Interface closure failed\n{delete_result.message}", show_alert=True
            )
            return None
        await message.answer("✅ Interface closed successfully", show_alert=True)
        return None

    user_id = message.sender_user_id
    user = await c.getUser(user_id)
    if isinstance(user, types.Error):
        c.logger.warning(f"Failed to get user info: {user.message}")
        return None

    user_name = user.first_name
    # Handle music playback requests
    try:
        _, platform, song_id = data.split("_", 2)
    except ValueError:
        c.logger.error(f"Malformed callback data received: {data}")
        await message.answer("⚠️ Invalid request format", show_alert=True)
        return None

    await message.answer(f"🔍 Preparing playback for {user_name}", show_alert=True)
    reply = await message.edit_message_text(
        f"🔍 Searching...\nRequested by: {user_name}"
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Message edit failed: {reply.message}")
        return None

    url = _get_platform_url(platform, song_id)
    if not url:
        c.logger.error(f"Unsupported platform: {platform} | Data: {data}")
        await edit_text(reply, text=f"⚠️ Unsupported platform: {platform}")
        return None

    song = await DownloaderWrapper(url).get_info()
    if song:
        if isinstance(song, types.Error):
            await edit_text(reply, text=f"⚠️ Retrieval error\n{song.message}")
            return None

        return await play_music(c, reply, song, user_name)

    await edit_text(reply, text="⚠️ Requested content not found")
    return None


@Client.on_updateNewCallbackQuery(filters=Filter.regex(r"play_\w+"))
@admins_only(is_bot=True, is_auth=True)
async def callback_query(c: Client, message: types.UpdateNewCallbackQuery) -> None:
    """Handles all playback control callback queries.

    This function acts as a central dispatcher for callbacks from the player
    control buttons (e.g., skip, stop, pause, resume). It ensures the user
    is authorized and the bot is in an active call before processing the action.

    Args:
        c (Client): The pytdbot client instance.
        message (types.UpdateNewCallbackQuery): The callback query update.
    """
    data = message.payload.data.decode()
    user_id = message.sender_user_id

    # Retrieve message and user info with error handling
    get_msg = await message.getMessage()
    if isinstance(get_msg, types.Error):
        c.logger.warning(f"Failed to get message: {get_msg.message}")
        return None

    user = await c.getUser(user_id)
    if isinstance(user, types.Error):
        c.logger.warning(f"Failed to get user info: {user.message}")
        return None

    user_name = user.first_name

    def requires_active_chat(action: str) -> bool:
        """Check if action requires an active playback session."""
        return action in {
            "play_skip",
            "play_stop",
            "play_pause",
            "play_resume",
        }

    async def send_response(
        msg: str, alert: bool = False, delete: bool = False, reply_markup=None
    ) -> None:
        """Helper function to send standardized responses."""
        if alert:
            await message.answer(msg, show_alert=True)
        else:
            edit_func = (
                message.edit_message_caption
                if get_msg.caption
                else message.edit_message_text
            )
            await edit_func(msg, reply_markup=reply_markup)

        if delete:
            _del_result = await c.deleteMessages(
                message.chat_id, [message.message_id], revoke=True
            )
            if isinstance(_del_result, types.Error):
                c.logger.warning(f"Message deletion failed: {_del_result.message}")

    chat_id = message.chat_id
    if requires_active_chat(data) and not chat_cache.is_active(chat_id):
        return await send_response(
            "⏹️ No active playback session in this chat.", alert=True
        )

    # Handle different control actions
    if data == "play_skip":
        result = await call.play_next(chat_id)
        if isinstance(result, types.Error):
            return await send_response(
                f"⚠️ Playback error\nDetails: {result.message}",
                alert=True,
            )
        return await send_response("⏭️ Track skipped successfully", delete=True)

    if data == "play_stop":
        result = await call.end(chat_id)
        if isinstance(result, types.Error):
            return await send_response(
                f"⚠️ Failed to stop playback\n{result.message}", alert=True
            )
        return await send_response(
            f"<b>⏹ Playback Stopped</b>\n└ Requested by: {user_name}"
        )

    if data == "play_pause":
        result = await call.pause(chat_id)
        if isinstance(result, types.Error):
            return await send_response(
                f"⚠️ Pause failed\n{result.message}",
                alert=True,
            )
        markup = (
            control_buttons("pause") if await db.get_buttons_status(chat_id) else None
        )
        return await send_response(
            f"<b>⏸ Playback Paused</b>\n└ Requested by: {user_name}",
            reply_markup=markup,
        )

    if data == "play_resume":
        result = await call.resume(chat_id)
        if isinstance(result, types.Error):
            return await send_response(f"⚠️ Resume failed\n{result.message}", alert=True)
        markup = (
            control_buttons("resume") if await db.get_buttons_status(chat_id) else None
        )
        return await send_response(
            f"<b>▶ Playback Resumed</b>\n└ Requested by: {user_name}",
            reply_markup=markup,
        )

    if data.startswith("play_c_"):
        return await _handle_play_c_data(data, message, chat_id, user_id, user_name, c)
    return None
