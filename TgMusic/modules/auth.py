# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, db
from TgMusic.logger import LOGGER


async def _validate_auth_command(msg: types.Message) -> Union[types.Message, None]:
    """Validate authorization command requirements."""
    chat_id = msg.chat_id
    if chat_id > 0:
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
@admins_only(permissions="can_manage_chat", is_both=True)
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
@admins_only(permissions="can_manage_chat", is_both=True)
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
@admins_only(permissions="can_manage_chat", is_both=True)
async def auth_list(c: Client, msg: types.Message) -> None:
    """List all authorized users."""
    chat_id = msg.chat_id
    if chat_id > 0:
        reply = await msg.reply_text("❌ This command is only available in groups.")
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
