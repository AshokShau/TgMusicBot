# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union

from pytdbot import Client, types

from TgMusic.core import Filter, db
from TgMusic.core.admins import is_admin, is_owner
from TgMusic.logger import LOGGER
from TgMusic.modules.utils.play_helpers import extract_argument


async def _validate_auth_command(msg: types.Message) -> Union[types.Message, None]:
    """Validate authorization command requirements."""
    chat_id = msg.chat_id
    if chat_id > 0:
        return None

    if not await is_owner(chat_id, msg.from_id):
        reply = await msg.reply_text("⛔ Only the group owner can use this command.")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return None

    if not msg.reply_to_message_id:
        reply = await msg.reply_text(
            "🔍 Please reply to a user to manage their permissions."
        )
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return None

    reply = await msg.getRepliedMessage()
    if isinstance(reply, types.Error):
        reply = await msg.reply_text(f"⚠️ Error: {reply.message}")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return None

    if reply.from_id == msg.from_id:
        _reply = await msg.reply_text("❌ You cannot modify your own permissions.")
        if isinstance(_reply, types.Error):
            LOGGER.warning(_reply.message)
        return None

    if isinstance(reply.sender_id, types.MessageSenderChat):
        _reply = await msg.reply_text("❌ Channels cannot be granted user permissions.")
        if isinstance(_reply, types.Error):
            LOGGER.warning(_reply.message)
        return None

    return reply


@Client.on_message(filters=Filter.command(["auth"]))
async def auth(c: Client, msg: types.Message) -> None:
    """Grant authorization permissions to a user."""
    reply = await _validate_auth_command(msg)
    if not reply:
        return

    chat_id = msg.chat_id
    user_id = reply.from_id

    if user_id in await db.get_auth_users(chat_id):
        reply = await msg.reply_text("ℹ️ User already has authorization permissions.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
    else:
        await db.add_auth_user(chat_id, user_id)
        reply = await msg.reply_text(
            "✅ User successfully granted authorization permissions."
        )
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)


@Client.on_message(filters=Filter.command(["unauth"]))
async def un_auth(c: Client, msg: types.Message) -> None:
    """Revoke authorization permissions from a user."""
    reply = await _validate_auth_command(msg)
    if not reply:
        return

    chat_id = msg.chat_id
    user_id = reply.from_id

    if user_id not in await db.get_auth_users(chat_id):
        reply = await msg.reply_text("ℹ️ User doesn't have authorization permissions.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
    else:
        await db.remove_auth_user(chat_id, user_id)
        reply = await msg.reply_text(
            "✅ User's authorization permissions have been revoked."
        )
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)


@Client.on_message(filters=Filter.command(["authlist"]))
async def auth_list(c: Client, msg: types.Message) -> None:
    """List all authorized users."""
    chat_id = msg.chat_id
    if chat_id > 0:
        reply = await msg.reply_text("❌ This command is only available in groups.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    if not await is_admin(chat_id, msg.from_id):
        reply = await msg.reply_text("⛔ Administrator privileges required.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    auth_users = await db.get_auth_users(chat_id)
    if not auth_users:
        reply = await msg.reply_text("ℹ️ No authorized users found.")
        if isinstance(reply, types.Error):
            c.logger.warning(reply.message)
        return

    text = "<b>🔐 Authorized Users:</b>\n\n" + "\n".join(
        [f"• <code>{uid}</code>" for uid in auth_users]
    )
    reply = await msg.reply_text(text)
    if isinstance(reply, types.Error):
        c.logger.warning(reply.message)


@Client.on_message(filters=Filter.command(["buttons"]))
async def buttons(_: Client, msg: types.Message) -> None:
    """Toggle button controls."""
    chat_id = msg.chat_id
    if chat_id > 0:
        reply = await msg.reply_text("❌ This command is only available in groups.")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    if not await is_owner(chat_id, msg.from_id):
        reply = await msg.reply_text("⛔ Group owner privileges required.")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    current = await db.get_buttons_status(chat_id)
    args = extract_argument(msg.text)

    if not args:
        status = "enabled ✅" if current else "disabled ❌"
        reply = await msg.reply_text(
            f"⚙️ <b>Button Control Status:</b> {status}\n\n"
            "Usage: <code>/buttons [on|off|enable|disable]</code>"
        )
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    arg = args.lower()
    if arg in ["on", "enable"]:
        await db.set_buttons_status(chat_id, True)
        reply = await msg.reply_text("✅ Button controls enabled.")
    elif arg in ["off", "disable"]:
        await db.set_buttons_status(chat_id, False)
        reply = await msg.reply_text("❌ Button controls disabled.")
    else:
        reply = await msg.reply_text(
            "⚠️ Invalid command usage.\n"
            "Correct usage: <code>/buttons [enable|disable|on|off]</code>"
        )
    if isinstance(reply, types.Error):
        LOGGER.warning(reply.message)


@Client.on_message(filters=Filter.command(["thumbnail", "thumb"]))
async def thumbnail(_: Client, msg: types.Message) -> None:
    """Toggle thumbnail settings."""
    chat_id = msg.chat_id
    if chat_id > 0:
        reply = await msg.reply_text("❌ This command is only available in groups.")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    if not await is_owner(chat_id, msg.from_id):
        reply = await msg.reply_text("⛔ Group owner privileges required.")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    current = await db.get_thumb_status(chat_id)
    args = extract_argument(msg.text)

    if not args:
        status = "enabled ✅" if current else "disabled ❌"
        reply = await msg.reply_text(
            f"🖼️ <b>Thumbnail Status:</b> {status}\n\n"
            "Usage: <code>/thumbnail [on|off|enable|disable]</code>"
        )
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return

    arg = args.lower()
    if arg in ["on", "enable"]:
        await db.set_thumb_status(chat_id, True)
        reply = await msg.reply_text("✅ Thumbnails enabled.")
    elif arg in ["off", "disable"]:
        await db.set_thumb_status(chat_id, False)
        reply = await msg.reply_text("❌ Thumbnails disabled.")
    else:
        reply = await msg.reply_text(
            "⚠️ Invalid command usage.\n"
            "Correct usage: <code>/thumbnail [enable|disable|on|off]</code>"
        )
    if isinstance(reply, types.Error):
        LOGGER.warning(reply.message)
