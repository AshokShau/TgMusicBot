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
    def __init__(self, bot: Client):
        self.bot = bot
        self._stop = asyncio.Event()
        self._vc_task: asyncio.Task | None = None
        self._leave_task: asyncio.Task | None = None
        self._sleep_time = 40

    async def _end_call_if_inactive(self, chat_id: int) -> bool:
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
        """Start the background tasks."""
        self._stop.clear()
        if not config.NO_UPDATES:
            self._vc_task = asyncio.create_task(self._vc_loop())
            self.bot.logger.info("Started VC auto-end task")

        if config.AUTO_LEAVE:
            self._leave_task = asyncio.create_task(self._leave_loop())
            self.bot.logger.info("Started auto-leave task")

    async def stop(self):
        self._stop.set()

        if self._vc_task:
            await self._vc_task

        if self._leave_task:
            await self._leave_task

        self.bot.logger.info("All background loops stopped.")
