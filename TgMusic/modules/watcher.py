#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio

from pytdbot import Client, types

from TgMusic.core import (
    ChatMemberStatus,
    SupportButton,
    call,
    chat_cache,
    chat_invite_cache,
    config,
    db,
    load_admin_cache,
    user_status_cache,
)
from TgMusic.core.buttons import add_me_markup
from TgMusic.logger import LOGGER


async def handle_non_supergroup(client: Client, chat_id: int) -> None:
    """Handles the case where the bot is added to a non-supergroup chat.

    It sends an explanatory message and then leaves the chat.

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the non-supergroup chat.
    """
    text = (
        f"This chat ({chat_id}) is not a supergroup yet.\n"
        "<b>⚠️ Please convert this chat to a supergroup and add me as admin.</b>\n\n"
        "If you don't know how to convert, use this guide:\n"
        "🔗 https://te.legra.ph/How-to-Convert-a-Group-to-a-Supergroup-01-02\n\n"
        "If you have any questions, join our support group:"
    )
    bot_username = client.me.usernames.editable_username
    await client.sendTextMessage(
        chat_id=chat_id,
        text=text,
        reply_markup=add_me_markup(bot_username),
        disable_web_page_preview=True,
    )
    await asyncio.sleep(1)
    await client.leaveChat(chat_id)


def is_valid_supergroup(chat_id: int) -> bool:
    """Checks if a chat ID corresponds to a supergroup.

    Supergroup IDs in tdlib are represented as large negative numbers
    that typically start with "-100".

    Args:
        chat_id (int): The chat ID to check.

    Returns:
        bool: True if the chat is likely a supergroup, False otherwise.
    """
    return str(chat_id).startswith("-100")


async def handle_bot_join(client: Client, chat_id: int) -> None:
    """Handles the logic for when the bot joins a new chat.

    This includes checking the member count against the configured minimum
    and caching the chat's invite link if available.

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat the bot joined.
    """
    _chat_id = int(str(chat_id)[4:]) if str(chat_id).startswith("-100") else chat_id
    chat_info = await client.getSupergroupFullInfo(_chat_id)

    if isinstance(chat_info, types.Error):
        client.logger.warning(
            "Failed to get supergroup info for %s, %s", chat_id, chat_info.message
        )
        return

    if chat_info.member_count < config.MIN_MEMBER_COUNT:
        text = (
            f"⚠️ This group has too few members ({chat_info.member_count}).\n\n"
            "To prevent spam and ensure proper functionality, "
            f"this bot only works in groups with at least {config.MIN_MEMBER_COUNT} members.\n"
            "Please grow your community and add me again later.\n"
            "If you have any questions, join our support group:"
        )
        await client.sendTextMessage(chat_id, text, reply_markup=SupportButton)
        await asyncio.sleep(1)
        await client.leaveChat(chat_id)
        await db.remove_chat(chat_id)
        client.logger.info(
            "Bot left chat %s due to insufficient members (only %d present).",
            chat_id,
            chat_info.member_count,
        )
        return

    if invite_link := getattr(chat_info.invite_link, "invite_link", None):
        chat_invite_cache[chat_id] = invite_link


@Client.on_updateChatMember()
async def chat_member(client: Client, update: types.UpdateChatMember) -> None:
    """The main handler for chat member status updates.

    This function is triggered whenever a user joins, leaves, is promoted,
    or their status otherwise changes in a chat. It routes the update to
    more specific handler functions.

    Args:
        client (Client): The pytdbot client instance.
        update (types.UpdateChatMember): The chat member update object.
    """
    chat_id = update.chat_id

    # Early return for non-group chats
    if chat_id > 0 or not await _validate_chat(client, chat_id):
        return None

    await db.add_chat(chat_id)
    new_member = update.new_chat_member.member_id
    user_id = (
        new_member.user_id
        if isinstance(new_member, types.MessageSenderUser)
        else new_member.chat_id
    )
    old_status = update.old_chat_member.status["@type"]
    new_status = update.new_chat_member.status["@type"]

    # Handle different status change scenarios
    await _handle_status_changes(client, chat_id, user_id, old_status, new_status)
    return None


async def _validate_chat(client: Client, chat_id: int) -> bool:
    """Validates if a chat is a supergroup and handles non-supergroups.

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat to validate.

    Returns:
        bool: True if the chat is a valid supergroup, False otherwise.
    """
    if not is_valid_supergroup(chat_id):
        await handle_non_supergroup(client, chat_id)
        return False
    return True


async def _handle_status_changes(
    client: Client, chat_id: int, user_id: int, old_status: str, new_status: str
) -> None:
    """Routes a chat member update to the appropriate specific handler.

    This function acts as a router based on the change in the user's status
    (e.g., from "left" to "member", or to "banned").

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat where the update occurred.
        user_id (int): The ID of the user whose status changed.
        old_status (str): The user's previous status type name.
        new_status (str): The user's new status type name.
    """
    if old_status == "chatMemberStatusLeft" and new_status in {
        "chatMemberStatusMember",
        "chatMemberStatusAdministrator",
    }:
        await _handle_join(client, chat_id, user_id)
    elif (
        old_status in {"chatMemberStatusMember", "chatMemberStatusAdministrator"}
        and new_status == "chatMemberStatusLeft"
    ):
        await _handle_leave_or_kick(chat_id, user_id)
    elif new_status == "chatMemberStatusBanned":
        if user_id == client.me.id:
            await call.end(chat_id)
        await _handle_ban(chat_id, user_id)
    elif (
        old_status == "chatMemberStatusBanned" and new_status == "chatMemberStatusLeft"
    ):
        await _handle_unban(chat_id, user_id)
    else:
        await _handle_promotion_demotion(
            client, chat_id, user_id, old_status, new_status
        )


async def _handle_join(client: Client, chat_id: int, user_id: int) -> None:
    """Handles the logic for a user or the bot joining a chat.

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who joined.
    """
    if user_id == client.options["my_id"]:
        await handle_bot_join(client, chat_id)
    LOGGER.debug("User %s joined the chat %s.", user_id, chat_id)


async def _handle_leave_or_kick(chat_id: int, user_id: int) -> None:
    """Handles a user leaving or being kicked from a chat.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who left or was kicked.
    """
    LOGGER.debug("User %s left or was kicked from %s.", user_id, chat_id)
    await _update_user_status_cache(chat_id, user_id, types.ChatMemberStatusLeft())


async def _handle_ban(chat_id: int, user_id: int) -> None:
    """Handles a user being banned from a chat.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who was banned.
    """
    LOGGER.debug("User %s was banned in %s.", user_id, chat_id)
    await _update_user_status_cache(chat_id, user_id, types.ChatMemberStatusBanned())


async def _handle_unban(chat_id: int, user_id: int) -> None:
    """Handles a user being unbanned from a chat.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who was unbanned.
    """
    LOGGER.debug("User %s was unbanned in %s.", user_id, chat_id)
    await _update_user_status_cache(chat_id, user_id, types.ChatMemberStatusLeft())


async def _handle_promotion_demotion(
    client: Client, chat_id: int, user_id: int, old_status: str, new_status: str
) -> None:
    """Handles a user being promoted to or demoted from admin status.

    This triggers a reload of the admin cache for the chat.

    Args:
        client (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user who was promoted or demoted.
        old_status (str): The user's previous status type name.
        new_status (str): The user's new status type name.
    """
    is_promoted = (
        old_status != "chatMemberStatusAdministrator"
        and new_status == "chatMemberStatusAdministrator"
    )
    is_demoted = (
        old_status == "chatMemberStatusAdministrator"
        and new_status != "chatMemberStatusAdministrator"
    )

    if not (is_promoted or is_demoted):
        return

    if user_id == client.options["my_id"] and is_promoted:
        LOGGER.info("Bot promoted in %s. Reloading admin cache.", chat_id)
    else:
        action = "promoted" if is_promoted else "demoted"
        LOGGER.debug("User %s was %s in %s.", user_id, action, chat_id)

    await load_admin_cache(client, chat_id, True)
    await asyncio.sleep(1)
    if is_promoted:
        await handle_bot_join(client, chat_id)


async def _update_user_status_cache(
    chat_id: int, user_id: int, status: ChatMemberStatus
) -> None:
    """Updates the user status cache, specifically for the assistant client.

    This is important for keeping track of the assistant's membership status
    to avoid unnecessary join attempts.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user whose status changed.
        status (ChatMemberStatus): The new status object for the user.
    """
    ub = await call.get_client(chat_id)
    if isinstance(ub, types.Error):
        LOGGER.warning("Error getting client for chat %s: %s", chat_id, ub)
        return

    if user_id == ub.me.id:
        cache_key = f"{chat_id}:{ub.me.id}"
        user_status_cache[cache_key] = status


@Client.on_updateNewMessage(position=1)
async def new_message(client: Client, update: types.UpdateNewMessage) -> None:
    """Handles `updateNewMessage` events, primarily for service messages.

    This function watches for messages indicating that a video chat has
    started or ended, and takes appropriate action like clearing the queue
    and notifying the chat. It also adds new chats and users to the database.

    Args:
        client (Client): The pytdbot client instance.
        update (types.UpdateNewMessage): The new message update object.
    """
    message = update.message
    if not message:
        return

    chat_id = message.chat_id
    content = message.content

    # Run DB operation in the background
    if chat_id < 0:
        client.loop.create_task(db.add_chat(chat_id))
    else:
        client.loop.create_task(db.add_user(chat_id))

    # Handle video chat events
    if isinstance(content, types.MessageVideoChatEnded):
        LOGGER.info("Video chat ended in %s", chat_id)
        chat_cache.clear_chat(chat_id)
        await client.sendTextMessage(chat_id, "Video chat ended!\nAll queues cleared")
        return

    if isinstance(content, types.MessageVideoChatStarted):
        LOGGER.info("Video chat started in %s", chat_id)
        await call.end(chat_id)
        chat_cache.clear_chat(chat_id)
        await client.sendTextMessage(
            chat_id, "Video chat started!\nUse /play song name to play a song"
        )
        return

    LOGGER.debug("New message in %s: %s", chat_id, message)
