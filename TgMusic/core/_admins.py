# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

from functools import partial, wraps
from typing import Any, Callable, List, Literal, Optional, Tuple, Union

from cachetools import TTLCache
from pytdbot import Client, types

from ._config import config
from ._database import db
from ._filters import Filter

admin_cache = TTLCache(maxsize=1000, ttl=60 * 60)

ChatAdminPermissions = Literal[
    "can_manage_chat",
    "can_change_info",
    "can_delete_messages",
    "can_invite_users",
    "can_restrict_members",
    "can_pin_messages",
    "can_promote_members",
    "can_manage_video_chats",
]

PermissionsType = Union[ChatAdminPermissions, List[ChatAdminPermissions], None]


class AdminCache:
    def __init__(
        self, chat_id: int, user_info: list[types.ChatMember], cached: bool = True
    ):
        self.chat_id = chat_id
        self.user_info = user_info
        self.cached = cached


async def load_admin_cache(
    c: Client, chat_id: int, force_reload: bool = False
) -> Tuple[bool, AdminCache]:
    """
    Load the admin list from Telegram and cache it, unless already cached.
    Set force_reload to True to bypass the cache and reload the admin list.
    """
    if not force_reload and chat_id in admin_cache:
        return True, admin_cache[chat_id]

    admin_list = await c.searchChatMembers(
        chat_id, filter=types.ChatMembersFilterAdministrators()
    )
    if isinstance(admin_list, types.Error):
        c.logger.warning(
            f"Error loading admin cache for chat_id {chat_id}: {admin_list}"
        )
        return False, AdminCache(chat_id, [], cached=False)

    admin_cache[chat_id] = AdminCache(chat_id, admin_list["members"])
    return True, admin_cache[chat_id]


async def get_admin_cache_user(
    chat_id: int, user_id: int
) -> Tuple[bool, Optional[dict]]:
    """
    Check if the user is an admin using cached data.
    """
    admin_list = admin_cache.get(chat_id)
    if admin_list is None:
        return False, None

    return next(
        (
            (True, user_info)
            for user_info in admin_list.user_info
            if user_info["member_id"]["user_id"] == user_id
        ),
        (False, None),
    )


ANON = TTLCache(maxsize=250, ttl=60)


def ensure_permissions_list(permissions: PermissionsType) -> List[ChatAdminPermissions]:
    """Ensures permissions are a list of strings."""
    if permissions is None:
        return []
    return [permissions] if isinstance(permissions, str) else permissions


async def check_permissions(
    chat_id: int, user_id: int, permissions: PermissionsType
) -> bool:
    """
    Check if a user has specific permissions.
    """
    if not await is_admin(chat_id, user_id):
        return False

    if await is_owner(chat_id, user_id):
        return True

    permissions_list = ensure_permissions_list(permissions)
    if not permissions_list:
        return True

    _, user_info = await get_admin_cache_user(chat_id, user_id)
    if not user_info:
        return False

    rights = user_info["status"]["rights"]
    return all(getattr(rights, perm, False) for perm in permissions_list)


async def is_owner(chat_id: int, user_id: int) -> bool:
    """
    Check if the user is the owner of the chat.
    """
    is_cached, user = await get_admin_cache_user(chat_id, user_id)
    if not user:
        return False
    user_status = user["status"]["@type"]
    return is_cached and user_status == "chatMemberStatusCreator"


async def is_admin(chat_id: int, user_id: int) -> bool:
    """
    Check if the user is an admin in the chat.
    """
    is_cached, user = await get_admin_cache_user(chat_id, user_id)
    if not user:
        return False
    if chat_id == user_id:
        return True  # Anon Admin

    user_status = user["status"]["@type"]
    return is_cached and user_status in [
        "chatMemberStatusCreator",
        "chatMemberStatusAdministrator",
    ]


@Client.on_updateNewCallbackQuery(filters=Filter.regex("^anon."))
async def verify_anonymous_admin(
    c: Client, callback: types.UpdateNewCallbackQuery
) -> None:
    """Verify anonymous admin permissions."""
    data = callback.payload.data.decode()
    chat_id = callback.chat_id
    callback_id = int(f"{chat_id}{data.split('.')[1]}")
    if callback_id not in ANON:
        await callback.edit_message_text("Button has expired")
        return

    message, func, permissions = ANON.pop(callback_id)
    if not message:
        await callback.answer("Failed to retrieve message", show_alert=True)
        return

    if not await check_permissions(
        message.chat.id, callback.sender_user_id, permissions
    ):
        await callback.answer(
            f"You lack required permissions: {', '.join(ensure_permissions_list(permissions))}",
            show_alert=True,
        )
        return

    await c.deleteMessages(message.chat.id, [callback.message_id])
    await func(c, message)


def admins_only(
    permissions: PermissionsType = None,
    is_bot: bool = False,
    is_auth: bool = False,
    is_user: bool = False,
    is_both: bool = False,
    only_owner: bool = False,
    only_dev: bool = False,
    allow_pm: bool = True,
    no_reply: bool = False,
) -> Callable[[Callable[..., Any]], Callable[..., Any]]:
    """
    Decorator to check if the user is an admin before executing the command.
    """

    def decorator(func: Callable[..., Any]) -> Callable[..., Any]:
        @wraps(func)
        async def wrapper(
            c: Client,
            message: Union[types.UpdateNewCallbackQuery, types.Message],
            *args,
            **kwargs,
        ) -> Optional[Any]:
            if message is None:
                c.logger.warning("msg is none")
                return None

            if isinstance(message, types.UpdateNewCallbackQuery):
                sender = partial(message.answer, show_alert=True)
                user_id = message.sender_user_id
                msg_id = message.message_id
                is_anonymous = False
            else:
                sender = message.reply_text
                msg_id = message.id
                user_id = message.from_id
                is_anonymous = message.sender_id and isinstance(
                    message.sender_id, types.MessageSenderChat
                )

            chat_id = message.chat_id

            if only_dev and user_id != config.OWNER_ID:
                if no_reply:
                    return None
                return await sender("Only developers can use this command.")

            if not allow_pm and chat_id < 0:
                if no_reply:
                    return None
                return await sender("This command can only be used in groups.")

            # Handle anonymous admins
            if is_anonymous and not no_reply:
                ANON[int(f"{chat_id}{msg_id}")] = (message, func, permissions)
                _type = types.InlineKeyboardButtonTypeCallback(
                    f"anon.{msg_id}".encode()
                )

                keyboard = types.ReplyMarkupInlineKeyboard(
                    [[types.InlineKeyboardButton(text="Verify Admin", type=_type)]]
                )

                return await message.reply_text(
                    "Please verify that you are an admin to perform this action.",
                    reply_markup=keyboard,
                )

            load, _ = await load_admin_cache(c, chat_id)
            if not load:
                if no_reply:
                    return None
                return await sender("I need to be an admin to do this.")

            if only_owner and not await is_owner(chat_id, user_id):
                if no_reply:
                    return None
                return await sender("Only the chat owner can use this command.")

            async def check_and_notify(
                subject_id: int, subject_name: str
            ) -> Optional[bool]:
                if not await is_admin(chat_id, subject_id):
                    if no_reply:
                        return None
                    await sender(f"{subject_name} needs to be an admin.")
                    return False

                if not await check_permissions(chat_id, subject_id, permissions):
                    if no_reply:
                        return None
                    await sender(
                        f"{subject_name} lacks required permissions: {', '.join(ensure_permissions_list(permissions))}."
                    )
                    return False
                return True

            if is_bot and not await check_and_notify(c.me.id, "I"):
                return None

            if is_user and not await check_and_notify(user_id, "You"):
                return None

            if is_auth:
                auth_users = await db.get_auth_users(chat_id)
                is_admin_user = await is_admin(chat_id, user_id)
                is_authorized = user_id in auth_users if auth_users else False
                if not (is_admin_user or is_authorized):
                    if no_reply:
                        return None
                    await sender(
                        "You need to be either an admin or an authorized user to use this command."
                    )
                    return None

            if is_both and (
                not await check_and_notify(user_id, "You")
                or not await check_and_notify(c.me.id, "I")
            ):
                return None

            return await func(c, message, *args, **kwargs)

        return wrapper

    return decorator
