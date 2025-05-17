#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Union, TypeAlias

import pyrogram
from cachetools import TTLCache
from pyrogram import errors
from pytdbot import Client, types

ChatMemberStatus: TypeAlias = Union[
    types.ChatMemberStatusCreator,
    types.ChatMemberStatusAdministrator,
    types.ChatMemberStatusMember,
    types.ChatMemberStatusRestricted,
    types.ChatMemberStatusLeft,
    types.ChatMemberStatusBanned,
]

ChatMemberStatusResult: TypeAlias = Union[ChatMemberStatus, types.Error]
user_status_cache: TTLCache[str, ChatMemberStatus] = TTLCache(maxsize=5000, ttl=1000)

chat_invite_cache = TTLCache(maxsize=1000, ttl=1000)

async def check_user_status(c: Client, chat_id: int, user_id: int) -> ChatMemberStatusResult:
    cache_key = f"{chat_id}:{user_id}"
    user_status = user_status_cache.get(cache_key)
    if not user_status:
        user = await c.getChatMember(chat_id=chat_id, member_id=types.MessageSenderUser(user_id))
        if isinstance(user, types.Error):
            if user.code == 400:
                return types.ChatMemberStatusLeft()
            raise user

        if user.status is None:
            return types.ChatMemberStatusLeft()

        user_status = user.status
        user_status_cache[cache_key] = user_status

    return user_status

async def join_ub(chat_id: int, c: Client, ub: pyrogram.Client) -> Union[types.Ok, types.Error]:
    """
    Handles the userbot joining a chat via invite link or approval.
    """
    invite_link = chat_invite_cache.get(chat_id)
    if not invite_link:
        get_link = await c.createChatInviteLink(chat_id, name="TgMusicBot")
        if isinstance(get_link, types.Error):
            return get_link
        invite_link = get_link.invite_link

    if not invite_link:
        return types.Error(
            code=400, message=f"Failed to get invite link for chat {chat_id}"
        )

    chat_invite_cache[chat_id] = invite_link
    invite_link = invite_link.replace("https://t.me/+", "https://t.me/joinchat/")
    cache_key = f"{chat_id}:{ub.me.id}"
    try:
        await ub.join_chat(invite_link)
        user_status_cache[cache_key] = types.ChatMemberStatusMember()
        return types.Ok()
    except errors.InviteRequestSent:
        ok = await c.processChatJoinRequest(
            chat_id=chat_id, user_id=ub.me.id, approve=True
        )
        return ok if isinstance(ok, types.Error) else None
    except errors.UserAlreadyParticipant:
        user_status_cache[cache_key] = types.ChatMemberStatusMember()
        return types.Ok()
    except errors.InviteHashExpired:
        return types.Error(code=400, message=f"Invite link has expired or my assistant ({ub.me.id}) is banned from this group.")
    except Exception as e:
        return types.Error(code=400, message=f"Failed to join {ub.me.id}: {e}")
