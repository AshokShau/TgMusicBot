#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re
from typing import Union

from pytdbot import filters, types


class Filter:
    @staticmethod
    def _extract_text(event) -> str | None:
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
        """
        Filter for messages or callback queries matching a regex pattern.
        """
        compiled = re.compile(pattern)

        async def filter_func(_, event) -> bool:
            text = Filter._extract_text(event)
            return bool(compiled.search(text)) if text else False

        return filters.create(filter_func)
