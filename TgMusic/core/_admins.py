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
    """A class to cache administrator information for a chat.

    Attributes:
        chat_id (int): The ID of the chat.
        user_info (list[types.ChatMember]): A list of chat member objects
            representing the administrators.
        cached (bool): A flag indicating if the information is from the cache.
    """

    def __init__(
        self, chat_id: int, user_info: list[types.ChatMember], cached: bool = True
    ):
        """Initializes the AdminCache instance.

        Args:
            chat_id (int): The ID of the chat.
            user_info (list[types.ChatMember]): A list of chat member objects.
            cached (bool): Whether the data is from the cache. Defaults to True.
        """
        self.chat_id = chat_id
        self.user_info = user_info
        self.cached = cached


async def load_admin_cache(
    c: Client, chat_id: int, force_reload: bool = False
) -> Tuple[bool, AdminCache]:
    """Loads the admin list from Telegram and caches it.

    This function fetches the list of administrators for a given chat,
    stores it in a time-to-live (TTL) cache, and returns it. If the
    admin list is already in the cache, it returns the cached data
    unless `force_reload` is set to True.

    Args:
        c (Client): The pytdbot client instance.
        chat_id (int): The ID of the chat for which to load admins.
        force_reload (bool): If True, bypasses the cache and reloads
            the admin list from Telegram. Defaults to False.

    Returns:
        Tuple[bool, AdminCache]: A tuple containing a boolean indicating
            the success of the operation and an `AdminCache` object.
            The boolean is False if an error occurred.
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
    """Retrieves a user's admin information from the cache.

    This function checks the cached admin list for a specific chat to find
    a particular user.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user to look for.

    Returns:
        Tuple[bool, Optional[dict]]: A tuple where the first element is a
            boolean indicating if the user was found in the admin cache,
            and the second element is the user's information dictionary
            if found, otherwise None.
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
    """Ensures that the given permissions are in a list format.

    Args:
        permissions (PermissionsType): The permissions, which can be a single
            string, a list of strings, or None.

    Returns:
        List[ChatAdminPermissions]: A list of permission strings. Returns an
            empty list if the input is None.
    """
    if permissions is None:
        return []
    return [permissions] if isinstance(permissions, str) else permissions


async def check_permissions(
    chat_id: int, user_id: int, permissions: PermissionsType
) -> bool:
    """Checks if a user has a specific set of permissions in a chat.

    This function first verifies if the user is an admin. If they are, it
    checks if they hold all the specified permissions. Chat owners are
    always considered to have all permissions.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user to check.
        permissions (PermissionsType): The permission or list of permissions
            to check.

    Returns:
        bool: True if the user has all the specified permissions, False otherwise.
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
    """Checks if a user is the owner of a chat.

    This function relies on the cached admin data to determine if the user's
    status is 'chatMemberStatusCreator'.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user.

    Returns:
        bool: True if the user is the chat owner, False otherwise.
    """
    is_cached, user = await get_admin_cache_user(chat_id, user_id)
    if not user:
        return False
    user_status = user["status"]["@type"]
    return is_cached and user_status == "chatMemberStatusCreator"


async def is_admin(chat_id: int, user_id: int) -> bool:
    """Checks if a user is an administrator in a chat.

    This function uses the cached admin data to determine if the user has
    admin or owner status. It also handles the case of anonymous admins
    in the chat.

    Args:
        chat_id (int): The ID of the chat.
        user_id (int): The ID of the user.

    Returns:
        bool: True if the user is an admin, False otherwise.
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
    """Handles the verification callback for an anonymous admin.

    When an admin using an anonymous identity clicks a verification button,
    this handler checks their permissions and proceeds with the original
    function call if successful.

    Args:
        c (Client): The pytdbot client instance.
        callback (types.UpdateNewCallbackQuery): The callback query update.

    Returns:
        None
    """
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
    """A decorator to restrict command access to administrators.

    This decorator provides a flexible way to protect handlers by checking
    various admin-related conditions before executing the wrapped function.
    It can check for user permissions, bot permissions, ownership, and more.

    Args:
        permissions (PermissionsType): A single permission or a list of
            permissions required to execute the command. Defaults to None.
        is_bot (bool): If True, checks if the bot itself has admin rights
            and the specified permissions. Defaults to False.
        is_auth (bool): If True, allows both admins and authorized users
            (from the database) to use the command. Defaults to False.
        is_user (bool): If True, checks if the user invoking the command
            has admin rights and permissions. Defaults to False.
        is_both (bool): If True, checks if both the user and the bot have
            the required admin rights and permissions. Defaults to False.
        only_owner (bool): If True, restricts the command to the chat owner.
            Defaults to False.
        only_dev (bool): If True, restricts the command to the bot's owner
            (developer). Defaults to False.
        allow_pm (bool): If True, allows the command to be used in private
            messages. Defaults to True.
        no_reply (bool): If True, suppresses reply messages on permission
            failure. Defaults to False.

    Returns:
        Callable: The decorated function.
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
