#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re
from typing import Union

from pytdbot import filters, types


class Filter:
    """A collection of custom filters for pytdbot events.

    This class provides static methods to create filters for commands and
    regular expressions, simplifying the process of routing updates like

    messages and callback queries to their appropriate handlers.
    """

    @staticmethod
    def _extract_text(event) -> str | None:
        """Extracts text content from various pytdbot event types.

        This is a helper method to safely retrieve text from `Message`,
        `UpdateNewMessage`, or `UpdateNewCallbackQuery` objects.

        Args:
            event: The pytdbot event object.

        Returns:
            str | None: The extracted text or None if no text is found.
        """
        if isinstance(event, types.Message) and hasattr(event.content, "text"):
            return event.content.text.text
        if isinstance(event, types.UpdateNewMessage) and hasattr(event.message, "text"):
            return event.message.text
        if isinstance(event, types.UpdateNewCallbackQuery) and getattr(
            event, "payload", None
        ):
            return event.payload.data.decode(errors="ignore")
        return None

    @staticmethod
    def command(
        commands: Union[str, list[str]], prefixes: str = "/!"
    ) -> filters.Filter:
        """Creates a filter for bot commands.

        This filter matches messages that start with a specified prefix
        (e.g., '/', '!') followed by one of the given command strings.
        It also handles commands directed at a specific bot (e.g., /command@botname).

        Args:
            commands (Union[str, list[str]]): A command or list of commands to match.
            prefixes (str): A string of characters to be treated as command
                prefixes. Defaults to "/!".

        Returns:
            filters.Filter: A pytdbot filter instance.
        """
        if isinstance(commands, str):
            commands = [commands]
        commands_set = {cmd.lower() for cmd in commands}

        pattern = re.compile(
            rf"^[{re.escape(prefixes)}](\w+)(?:@(\w+))?", re.IGNORECASE
        )

        async def filter_func(client, event) -> bool:
            text = Filter._extract_text(event)
            if not text:
                return False

            match = pattern.match(text)
            if not match:
                return False

            cmd, mentioned_bot = match.groups()
            if cmd.lower() not in commands_set:
                return False

            if mentioned_bot:
                bot_username = getattr(client.me.usernames, "editable_username", None)
                return (
                    bool(bot_username) and mentioned_bot.lower() == bot_username.lower()
                )

            return True

        return filters.create(filter_func)

    @staticmethod
    def regex(pattern: str) -> filters.Filter:
        """Creates a filter for messages or callbacks matching a regex pattern.

        This filter checks if the text content of an event contains a match
        for the given regular expression.

        Args:
            pattern (str): The regular expression pattern to search for.

        Returns:
            filters.Filter: A pytdbot filter instance.
        """
        compiled = re.compile(pattern)

        async def filter_func(_, event) -> bool:
            text = Filter._extract_text(event)
            return bool(compiled.search(text)) if text else False

        return filters.create(filter_func)
