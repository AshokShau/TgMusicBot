#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache, db
from TgMusic.modules.utils.play_helpers import extract_argument


@Client.on_message(filters=Filter.command(["playtype", "setPlayType"]))
@admins_only(is_bot=True, is_auth=True)
async def set_play_type(_: Client, msg: types.Message) -> None:
    """Configures the playback mode for the chat.

    This command allows authorized users to choose between two modes:
    0: The bot immediately plays the first result of a search.
    1: The bot presents a list of search results for the user to choose from.

    Args:
        _ (Client): The pytdbot client instance (unused).
        msg (types.Message): The message object containing the command.
    """
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
    """Checks if a command can be executed by verifying an active playback session.

    Note: The name is slightly misleading. The `@admins_only` decorator handles
    the admin/auth check. This function's primary role is to ensure there's
    an active call in the chat before proceeding.

    Args:
        msg (types.Message): The message object to validate.

    Returns:
        Union[int, types.Message, types.Error]: The chat ID if the check passes,
            or a message/error object if there's no active session.
    """
    chat_id = msg.chat_id

    if not chat_cache.is_active(chat_id):
        return await msg.reply_text("⏸ No active playback session")

    return chat_id


async def handle_playback_action(
    c: Client, msg: types.Message, action, success_msg: str, fail_msg: str
) -> None:
    """A generic handler for simple playback control commands.

    This function abstracts the common pattern for commands like pause, resume,
    mute, and unmute. It validates the request, calls the appropriate action
    from the `call` object, and sends a success or failure message.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message that triggered the action.
        action: The callable action to perform (e.g., `call.pause`).
        success_msg (str): The message to send on success.
        fail_msg (str): The message to send on failure.
    """
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
    """Handles the /pause command to pause the current playback.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
    await handle_playback_action(
        c, msg, call.pause, "⏸ Playback paused", "Failed to pause playback"
    )


@Client.on_message(filters=Filter.command("resume"))
@admins_only(is_bot=True, is_auth=True)
async def resume(c: Client, msg: types.Message) -> None:
    """Handles the /resume command to resume paused playback.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
    await handle_playback_action(
        c, msg, call.resume, "▶️ Playback resumed", "Failed to resume playback"
    )


@Client.on_message(filters=Filter.command("mute"))
@admins_only(is_bot=True, is_auth=True)
async def mute_song(c: Client, msg: types.Message) -> None:
    """Handles the /mute command to mute the bot's audio in the call.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
    await handle_playback_action(
        c, msg, call.mute, "🔇 Audio muted", "Failed to mute audio"
    )


@Client.on_message(filters=Filter.command("unmute"))
@admins_only(is_bot=True, is_auth=True)
async def unmute_song(c: Client, msg: types.Message) -> None:
    """Handles the /unmute command to unmute the bot's audio in the call.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
    await handle_playback_action(
        c, msg, call.unmute, "🔊 Audio unmuted", "Failed to unmute audio"
    )
