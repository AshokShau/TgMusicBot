#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union
from pytdbot import Client, types

from TgMusic.core import Filter, chat_cache, call, db, admins_only
from TgMusic.modules.utils.play_helpers import extract_argument


@Client.on_message(filters=Filter.command(["playtype", "setPlayType"]))
@admins_only(is_bot=True, is_auth=True)
async def set_play_type(_: Client, msg: types.Message) -> None:
    """Configure playback mode."""
    chat_id = msg.chat_id
    if chat_id > 0:
        return

    play_type = extract_argument(msg.text, enforce_digit=True)
    if not play_type:
        text = "Usage: /setPlayType 0/1\n\n0 = Directly play the first search result.\n1 = Show a list of songs to choose from."
        await msg.reply_text(text)
        return

    play_type = int(play_type)
    if play_type not in (0, 1):
        await msg.reply_text("⚠️ Invalid mode. Use 0  or 1")
        return

    await db.set_play_type(chat_id, play_type)
    await msg.reply_text(f"🔀 Playback mode set to: <b>{play_type}</b>")


async def is_admin_or_reply(
    msg: types.Message,
) -> Union[int, types.Message, types.Error]:
    """Verify admin status and active playback session."""
    chat_id = msg.chat_id

    if not chat_cache.is_active(chat_id):
        return await msg.reply_text("⏸ No active playback session")

    return chat_id


async def handle_playback_action(
    c: Client, msg: types.Message, action, success_msg: str, fail_msg: str
) -> None:
    """Handle common playback control operations."""
    _chat_id = await is_admin_or_reply(msg)
    if isinstance(_chat_id, types.Error):
        c.logger.warning(f"⚠️ Admin check failed: {_chat_id.message}")
        return

    if isinstance(_chat_id, types.Message):
        return

    result = await action(_chat_id)
    if isinstance(result, types.Error):
        await msg.reply_text(f"⚠️ {fail_msg}\n<code>{result.message}</code>")
        return

    await msg.reply_text(f"{success_msg}\n" f"└ Requested by: {await msg.mention()}")


@Client.on_message(filters=Filter.command("pause"))
@admins_only(is_bot=True, is_auth=True)
async def pause_song(c: Client, msg: types.Message) -> None:
    """Pause current playback."""
    await handle_playback_action(
        c, msg, call.pause, "⏸ Playback paused", "Failed to pause playback"
    )


@Client.on_message(filters=Filter.command("resume"))
@admins_only(is_bot=True, is_auth=True)
async def resume(c: Client, msg: types.Message) -> None:
    """Resume paused playback."""
    await handle_playback_action(
        c, msg, call.resume, "▶️ Playback resumed", "Failed to resume playback"
    )


@Client.on_message(filters=Filter.command("mute"))
@admins_only(is_bot=True, is_auth=True)
async def mute_song(c: Client, msg: types.Message) -> None:
    """Mute audio playback."""
    await handle_playback_action(
        c, msg, call.mute, "🔇 Audio muted", "Failed to mute audio"
    )


@Client.on_message(filters=Filter.command("unmute"))
@admins_only(is_bot=True, is_auth=True)
async def unmute_song(c: Client, msg: types.Message) -> None:
    """Unmute audio playback."""
    await handle_playback_action(
        c, msg, call.unmute, "🔊 Audio unmuted", "Failed to unmute audio"
    )
