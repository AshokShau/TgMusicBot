# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.


import asyncio
from datetime import datetime

from pytdbot import Client, types

__version__ = "1.2.4"
StartTime = datetime.now()

from TgMusic.core import call, config, db, tg


class Bot(Client):
    """Main bot class handling initialization and lifecycle management."""

    def __init__(self) -> None:
        """Initialize the bot with configuration and services."""
        super().__init__(
            token=config.TOKEN,
            api_id=config.API_ID,
            api_hash=config.API_HASH,
            default_parse_mode="html",
            td_log=types.LogStreamEmpty(),
            plugins=types.plugins.Plugins(folder="TgMusic/modules"),
            files_directory="",
            database_encryption_key="",
            options={"ignore_background_updates": config.IGNORE_BACKGROUND_UPDATES},
        )
        self._initialize_services()

    def _initialize_services(self) -> None:
        """Initialize all service dependencies."""
        from TgMusic.modules.jobs import InactiveCallManager

        self.config = config
        self.db = db
        self.call = call
        self.tg = tg
        self.call_manager = InactiveCallManager(self)
        self._start_time = StartTime
        self._version = __version__

    async def start_clients(self) -> None:
        """Initialize all client sessions."""
        try:
            await asyncio.gather(
                *[
                    self.call.start_client(config.API_ID, config.API_HASH, session_str)
                    for session_str in config.SESSION_STRINGS
                ]
            )
        except Exception as exc:
            raise SystemExit(1) from exc

    async def initialize_components(self) -> None:
        from TgMusic.core import save_all_cookies

        await save_all_cookies(config.COOKIES_URL)
        await self.db.ping()
        await self.start_clients()
        await self.call.add_bot(self)
        await self.call.register_decorators()
        await super().start()
        await self.call_manager.start()
        uptime = self._get_uptime()
        self.logger.info(f"Bot started successfully in {uptime:.2f} seconds")
        self.logger.info(f"Version: {self._version}")

    async def stop_task(self) -> None:
        self.logger.info("Stopping bot...")
        try:
            shutdown_tasks = [
                self.db.close(),
                self.call_manager.stop(),
                self.call.stop_all_clients(),
            ]
            await asyncio.gather(*shutdown_tasks)
        except Exception as e:
            self.logger.error(f"Error during shutdown: {e}", exc_info=True)
            raise

    def _get_uptime(self) -> float:
        """Calculate bot uptime in seconds."""
        return (datetime.now() - self._start_time).total_seconds()


client: Bot = Bot()
