# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html

import asyncio
import time
from datetime import datetime, timedelta

from pyrogram import errors
from pyrogram.client import Client as PyroClient
from pytdbot import Client, types

from TgMusic.core import call, chat_cache, config, db


class InactiveCallManager:
    """Manages background jobs for the bot.

    This class handles two main background tasks:
    1.  **VC Auto-End**: Periodically checks active voice chats and ends the call
        if no one is listening, to conserve resources.
    2.  **Auto-Leave**: A daily task for assistant clients to leave all chats
        they are in, helping to keep the accounts clean.

    Attributes:
        bot (Client): The main pytdbot client instance.
    """

    def __init__(self, bot: Client):
        """Initializes the InactiveCallManager.

        Args:
            bot (Client): The main pytdbot client instance.
        """
        self.bot = bot
        self._stop = asyncio.Event()
        self._vc_task: asyncio.Task | None = None
        self._leave_task: asyncio.Task | None = None
        self._sleep_time = 40

    async def _end_call_if_inactive(self, chat_id: int) -> bool:
        """Checks a voice chat for inactivity and ends the call if needed.

        A call is considered inactive if there are no listeners (other than
        the assistant) for more than 15 seconds.

        Args:
            chat_id (int): The ID of the chat to check.

        Returns:
            bool: True if the call was ended, False otherwise.
        """
        vc_users = await call.vc_users(chat_id)
        if isinstance(vc_users, types.Error):
            self.bot.logger.warning(f"[VC Users Error] {chat_id}: {vc_users.message}")
            return False

        if len(vc_users) > 1:
            return False

        played_time = await call.played_time(chat_id)
        if isinstance(played_time, types.Error):
            self.bot.logger.warning(
                f"[Played Time Error] {chat_id}: {played_time.message}"
            )
            return False

        if played_time < 15:
            return False

        await self.bot.sendTextMessage(chat_id, "⚠️ No active listeners. Leaving VC...")
        await call.end(chat_id)
        return True

    async def _vc_loop(self):
        """The main loop for the voice chat auto-end feature.

        This loop runs continuously, checking all active voice chats at regular
        intervals and calling `_end_call_if_inactive` for each one.
        """
        while not self._stop.is_set():
            try:
                if self.bot.me is None:
                    await asyncio.sleep(2)
                    continue

                if not await db.get_auto_end(self.bot.me.id):
                    await asyncio.sleep(self._sleep_time)
                    continue

                active_chats = chat_cache.get_active_chats()
                if not active_chats:
                    await asyncio.sleep(self._sleep_time)
                    continue

                for chat_id in active_chats:
                    await self._end_call_if_inactive(chat_id)
                    await asyncio.sleep(0.1)

            except Exception as e:
                self.bot.logger.exception(f"[VC AutoEnd] Loop error: {e}")

            await asyncio.sleep(self._sleep_time)

    async def _leave_loop(self):
        """The main loop for the auto-leave feature.

        This loop is designed to run the `leave_all` method once every day
        at 3:00 AM.
        """
        while not self._stop.is_set():
            try:
                now = datetime.now()
                target = now.replace(hour=3, minute=0, second=0, microsecond=0)
                if now >= target:
                    target += timedelta(days=1)

                wait = (target - now).total_seconds()
                self.bot.logger.info(
                    f"[AutoLeave] Waiting {wait:.2f} seconds for 3:00 AM"
                )
                await asyncio.wait(
                    [asyncio.create_task(self._stop.wait())], timeout=wait
                )

                if self._stop.is_set():
                    break

                await self.leave_all()

                # Fallback safety sleep (24h)
                await asyncio.wait(
                    [asyncio.create_task(self._stop.wait())], timeout=86400
                )  # 24 hours
            except Exception as e:
                self.bot.logger.exception(f"[AutoLeave] Error: {e}")
                await asyncio.sleep(3600)  # Wait 1h before retry

    async def _leave_chat(self, ub: PyroClient, chat_id: int):
        """Makes a specific userbot (assistant) leave a chat.

        It includes handling for flood waits and other common RPC errors. It
        will not leave a chat if there is an active music session.

        Args:
            ub (PyroClient): The Pyrogram client instance of the assistant.
            chat_id (int): The ID of the chat to leave.

        Returns:
            bool: True if the chat was left successfully, False otherwise.
        """
        try:
            if chat_cache.is_active(chat_id):
                return False
            await ub.leave_chat(chat_id)
            self.bot.logger.debug(f"[{ub.name}] Left chat {chat_id}")
            return True
        except errors.FloodWait as e:
            wait_time = e.value
            if wait_time <= 100:
                self.bot.logger.warning(
                    f"[{ub.name}] FloodWait {wait_time}s for chat {chat_id}"
                )
                await asyncio.sleep(wait_time)
                return await self._leave_chat(ub, chat_id)
            return False
        except errors.RPCError as e:
            self.bot.logger.warning(f"[{ub.name}] RPCError on {chat_id}: {e}")
            return False
        except Exception as e:
            self.bot.logger.exception(f"[{ub.name}] Leave error on {chat_id}: {e}")
            return False

    async def leave_all(self):
        """Orchestrates the process of all assistants leaving all chats.

        This function iterates through each assistant client, gets its list of
        dialogs, and then calls `_leave_chat` for each group/channel.
        """
        if not config.AUTO_LEAVE:
            return

        self.bot.logger.info("[AutoLeave] Starting leave_all()")
        start_time = time.monotonic()

        try:
            for client_name, call_instance in call.calls.items():
                ub: PyroClient = call_instance.mtproto_client
                chats_to_leave = []

                try:
                    async for dialog in ub.get_dialogs():
                        chat = getattr(dialog, "chat", None)
                        if chat and chat.id > 0:
                            continue  # skip users/private chats
                        chats_to_leave.append(chat.id)
                except Exception as e:
                    self.bot.logger.exception(
                        f"[{client_name}] Failed to get dialogs: {e}"
                    )
                    continue

                self.bot.logger.info(
                    f"[{client_name}] Found {len(chats_to_leave)} chats to leave"
                )
                for chat_id in chats_to_leave:
                    await self._leave_chat(ub, chat_id)
                    await asyncio.sleep(0.5)

        except Exception as e:
            self.bot.logger.critical(f"[leave_all] Fatal error: {e}", exc_info=True)
        finally:
            duration = time.monotonic() - start_time
            self.bot.logger.info(f"[leave_all] Completed in {duration:.2f}s")

    async def start(self):
        """Starts the background job loops.

        This creates asyncio tasks for the VC auto-end loop and the auto-leave
        loop, respecting the bot's configuration settings.
        """
        self._stop.clear()
        if not config.NO_UPDATES:
            self._vc_task = asyncio.create_task(self._vc_loop())
            self.bot.logger.info("Started VC auto-end task")

        if config.AUTO_LEAVE:
            self._leave_task = asyncio.create_task(self._leave_loop())
            self.bot.logger.info("Started auto-leave task")

    async def stop(self):
        """Stops all running background tasks gracefully."""
        self._stop.set()

        if self._vc_task:
            await self._vc_task

        if self._leave_task:
            await self._leave_task

        self.bot.logger.info("All background loops stopped.")
