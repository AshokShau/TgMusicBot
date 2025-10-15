# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, db
from TgMusic.logger import LOGGER


async def _validate_auth_command(msg: types.Message) -> Union[types.Message, None]:
    """Validates the context for an authorization command.

    This helper function checks if the command was used correctly, for example:
    - It must be in a group chat.
    - It must be a reply to another user's message.
    - It cannot be a reply to oneself or a channel.

    Args:
        msg (types.Message): The message object that triggered the command.

    Returns:
        Union[types.Message, None]: The replied-to message object if validation
            passes, otherwise None. It sends appropriate error messages to the
            chat upon validation failure.
    """
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
    """Handles the /auth command to grant authorization to a user.

    This command allows an admin to add a user to the list of authorized
    users for the chat. An authorized user can use certain bot commands
    without being a chat admin.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
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
    """Handles the /unauth command to revoke a user's authorization.

    This command allows an admin to remove a user from the chat's list of
    authorized users.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
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
    """Handles the /authlist command to list all authorized users in the chat.

    Args:
        c (Client): The pytdbot client instance.
        msg (types.Message): The message object containing the command.
    """
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
