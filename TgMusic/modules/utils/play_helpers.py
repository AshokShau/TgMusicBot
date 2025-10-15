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
    """
    Extracts a URL from the given message or its reply.

    Args:
    msg: The message object to extract the URL from.
    reply: The reply message objects to extract the URL from, if any.

    Returns:
    The extracted URL string, or `None` if no URL was found.
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
    """
    Extracts the argument from the command text.

    Args:
        text (str): The full command text.
        enforce_digit (bool): Whether to enforce that the argument is a digit.

    Returns:
        str | None: The extracted argument or None if invalid.
    """
    args = text.strip().split(maxsplit=1)

    if len(args) < 2:
        return None

    argument = args[1].strip()
    return None if enforce_digit and not argument.isdigit() else argument


async def del_msg(msg: types.Message) -> None:
    """
    Deletes the given message.

    Args:
        msg (types.Message): The message to delete.

    Returns:
        None
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
    """
    Edits the given message and returns the result.

    If the given message is an Error, logs the error and returns it.
    If an exception occurs while editing the message, logs the exception and
    returns the original message.

    Args:
        reply_message (types.Message): The message to edit.
        *args: Passed to `Message.edit_text`.
        **kwargs: Passed to `Message.edit_text`.

    Returns:
        Union["types.Error", "types.Message"]: The edited message, or the
        original message if an exception occurred.
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
