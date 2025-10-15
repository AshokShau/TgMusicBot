#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
from typing import Any, Union

from pytdbot import types

from TgMusic.logger import LOGGER


async def get_url(
    msg: types.Message, reply: Union[types.Message, None]
) -> Union[str, None]:
    """Extracts a URL from a message's text or entities.

    This function checks the message entities first. If a URL entity is
    found, it's returned. It checks the replied-to message first if one
    is provided.

    Args:
        msg (types.Message): The original message object.
        reply (Union[types.Message, None]): The replied-to message object, if any.

    Returns:
        Union[str, None]: The extracted URL string, or None if no URL is found.
    """
    if reply:
        text_content = reply.text or ""
        entities = reply.entities or []
    else:
        text_content = msg.text or ""
        entities = msg.entities or []

    for entity in entities:
        if entity.type and entity.type["@type"] == "textEntityTypeUrl":
            offset = entity.offset
            length = entity.length
            return text_content[offset : offset + length]
    return None


def extract_argument(text: str, enforce_digit: bool = False) -> Union[str, None]:
    """Extracts the argument part of a command string.

    For example, in "/command arg1 arg2", it would return "arg1 arg2".

    Args:
        text (str): The full text of the message.
        enforce_digit (bool): If True, returns None unless the extracted
            argument consists only of digits. Defaults to False.

    Returns:
        Union[str, None]: The extracted argument string, or None if no
            argument is found or if `enforce_digit` is True and the
            argument is not a digit.
    """
    args = text.strip().split(maxsplit=1)

    if len(args) < 2:
        return None

    argument = args[1].strip()
    return None if enforce_digit and not argument.isdigit() else argument


async def del_msg(msg: types.Message) -> None:
    """Safely deletes a message, ignoring common errors.

    Args:
        msg (types.Message): The message object to delete.
    """
    delete = await msg.delete()
    if isinstance(delete, types.Error):
        if delete.code == 400:
            return
        LOGGER.warning("Error deleting message: %s", delete)
    return


async def edit_text(
    reply_message: types.Message, *args: Any, **kwargs: Any
) -> Union["types.Error", "types.Message"]:
    """A robust wrapper for editing a message's text.

    This function handles potential errors during the edit operation, such as
    the message not existing or being an error object itself. It also includes
    retry logic for flood wait errors.

    Args:
        reply_message (types.Message): The message object to be edited.
        *args: Positional arguments to be passed to `Message.edit_text`.
        **kwargs: Keyword arguments to be passed to `Message.edit_text`.

    Returns:
        Union["types.Error", "types.Message"]: The result of the edit
            operation, which could be the successfully edited message or an
            error object.
    """
    if isinstance(reply_message, types.Error):
        LOGGER.warning("Error getting message: %s", reply_message)
        return reply_message

    reply = await reply_message.edit_text(*args, **kwargs)
    if isinstance(reply, types.Error):
        if reply.code == 429:
            retry_after = (
                int(reply.message.split("retry after ")[1])
                if "retry after" in reply.message
                else 2
            )
            LOGGER.warning("Rate limited, retrying in %s seconds", retry_after)
            if retry_after > 20:
                return reply

            await asyncio.sleep(retry_after)
            return await edit_text(reply_message, *args, **kwargs)
        LOGGER.warning("Error editing message: %s", reply)
    return reply
