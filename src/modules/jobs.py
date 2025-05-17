#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio

from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.cron import CronTrigger
from pyrogram import Client as PyroClient
from pyrogram import errors
from pytdbot import Client, types

from src import db
from src.config import AUTO_LEAVE
from src.helpers import call
from src.helpers import chat_cache


class InactiveCallManager:
    def __init__(self, bot: Client):
        """
        Initialize the InactiveCallManager.

        Args:
            bot (Client): The client instance
        """
        self.bot = bot
        self.scheduler = AsyncIOScheduler(
            timezone="Asia/Kolkata", event_loop=self.bot.loop
        )
        self._lock = asyncio.Lock()

    async def _end_inactive_calls(self, chat_id: int, semaphore: asyncio.Semaphore):
        """
        End the call in a chat if there are no active listeners and the call has been
        active for more than 20 seconds.

        Args:
            chat_id (int): The chat ID to check for active listeners.
            semaphore (asyncio.Semaphore): A semaphore to limit concurrency.

        Returns:
            None
        """
        async with semaphore:
            vc_users = await call.vc_users(chat_id)
            if isinstance(vc_users, types.Error):
                self.bot.logger.warning(
                    f"An error occurred while getting vc users: {vc_users.message}"
                )
                return

            if len(vc_users) > 1:
                self.bot.logger.debug(
                    f"Active users detected in chat {chat_id}. Skipping..."
                )
                return

            # Check if the call has been active for more than 20 seconds
            played_time = await call.played_time(chat_id)
            if isinstance(played_time, types.Error):
                self.bot.logger.warning(
                    f"An error occurred while getting played time: {played_time.message}"
                )
                return
            if played_time < 20:
                self.bot.logger.debug(
                    f"Call in chat {chat_id} has been active for less than 20 "
                    "seconds. Skipping..."
                )
                return

            # Notify the chat and end the call
            _chat_id = await db.get_chat_id_by_channel(chat_id) or chat_id
            reply = await self.bot.sendTextMessage(
                _chat_id, "⚠️ No active listeners detected. ⏹️ Leaving voice chat..."
            )
            if isinstance(reply, types.Error):
                self.bot.logger.warning(f"Error sending message: {reply}")
            await call.end(chat_id)

    async def end_inactive_calls(self):
        """
        End calls in active chats with no listeners.

        This function retrieves active chats from the chat cache and, using semaphore to
        limit concurrency, checks each chat for inactive calls. It processes tasks in
        batches of 3 with a 1-second delay between batches to avoid overwhelming the
        system. If a call is inactive, it will be ended, and the chat will be notified.
        The function logs the number of active chats found and when inactive call checks
        are completed.
        """
        if self._lock.locked():
            self.bot.logger.warning("end_inactive_calls is already running, skipping...")
            return

        async with self._lock:
            if not await db.get_auto_end(self.bot.me.id):
                return
            active_chats = chat_cache.get_active_chats()
            self.bot.logger.debug(
                f"Found {len(active_chats)} active chats. Ending inactive calls..."
            )
            if not active_chats:
                return

            semaphore = asyncio.Semaphore(4)
            tasks = [
                self._end_inactive_calls(chat_id, semaphore) for chat_id in active_chats
            ]

            for i in range(0, len(tasks), 4):
                await asyncio.gather(*tasks[i: i + 4])
                await asyncio.sleep(0.5)

            self.bot.logger.debug("Inactive call checks completed.")

    async def leave_all(self):
        """
        Leave all chats for all userbot clients.

        This function iterates over all userbot clients and their associated chats. It
        skips private chats and active chats (i.e., chats with an ongoing call). For
        each non-active chat, it attempts to leave the chat using the associated userbot
        client. The function logs the number of chats found for each client, any
        FloodWait errors encountered, and any RPC errors encountered while attempting to
        leave chats. Finally, it logs when leaving all chats is completed for each
        client.

        Note that this function is intended to be used with caution, as it will leave
        all non-active chats for all userbot clients.
        """
        if not AUTO_LEAVE:
            return

        for client_name, call_instance in call.calls.items():
            ub: PyroClient = call_instance.mtproto_client
            chats_to_leave = []
            async for dialog in ub.get_dialogs():
                chat = getattr(dialog, "chat", None)
                if not chat:
                    continue
                if chat.id > 0:
                    self.bot.logger.debug(
                        f"[{client_name}] Skipping private chat: {chat.id}"
                    )
                    continue
                chats_to_leave.append(chat.id)
            self.bot.logger.debug(
                f"[{client_name}] Found {len(chats_to_leave)} chats to leave."
            )

            for chat_id in chats_to_leave:
                is_active = chat_cache.is_active(chat_id)
                if is_active:
                    continue
                try:
                    await ub.leave_chat(chat_id)
                    self.bot.logger.debug(f"[{client_name}] Left chat {chat_id}")
                    await asyncio.sleep(0.5)
                except errors.FloodWait as e:
                    wait_time = e.value
                    self.bot.logger.warning(
                        f"[{client_name}] FloodWait for {wait_time}s on chat {chat_id}"
                    )
                    if wait_time > 100:
                        self.bot.logger.warning(
                            f"[{client_name}] Skipping due to long wait time."
                        )
                        continue
                    await asyncio.sleep(wait_time)
                except errors.RPCError as e:
                    self.bot.logger.warning(
                        f"[{client_name}] Failed to leave chat {chat_id}: {e}"
                    )
                    continue
                except Exception as e:
                    self.bot.logger.error(
                        f"[{client_name}] Error leaving chat {chat_id}: {e}"
                    )
                    continue

            self.bot.logger.info(f"[{client_name}] Leaving all chats completed.")

    async def start_scheduler(self):
        """
        Start the scheduler.

        This function schedules two jobs to run at the specified intervals:
            - A job to end inactive calls every 50 seconds
            - A job to leave all non-active chats every day at 12:00 AM

        The scheduler is started after the jobs are added.

        Returns:
            None
        """
        self.scheduler.add_job(self.end_inactive_calls, "interval", seconds=50)
        self.scheduler.add_job(self.leave_all, CronTrigger(hour=0, minute=0))
        self.scheduler.start()
        self.bot.logger.info("Scheduler started.")

    async def stop_scheduler(self):
        """
        Stop the scheduler.

        This function stops the scheduler, causing all scheduled jobs to be
        unscheduled and not run again.

        Returns:
            None
        """
        self.scheduler.shutdown()
        self.bot.logger.info("Scheduler stopped.")
