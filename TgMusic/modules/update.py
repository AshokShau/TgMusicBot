#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import os
import shutil
import sys
import uuid
from os import execvp

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache
from TgMusic.logger import LOGGER


def is_docker():
    """Check if running inside a Docker container."""
    if os.path.exists("/.dockerenv"):
        return True
    if os.path.isfile("/proc/1/cgroup"):
        try:
            with open("/proc/1/cgroup", "r") as f:
                return "docker" in f.read()
        except Exception as e:
            LOGGER.warning("Failed to check if running in Docker: %s", e)
            return False
    return False


@Client.on_message(filters=Filter.command(["update", "restart"]))
@admins_only(only_dev=True)
async def update(c: Client, message: types.Message) -> None:
    """Handle /update and /restart commands."""
    command = message.text.strip().split()[0].lstrip("/")
    msg = await message.reply_text(
        f"{'Updating and ' if command == 'update' else ''}Restarting the bot..."
    )

    if command == "update":
        if not os.path.exists(".git"):
            await msg.edit_text(
                "⚠️ This instance does not support updates (no .git directory)."
            )
            return

        git_path = shutil.which("git") or "/usr/bin/git"
        if not os.path.isfile(git_path):
            await msg.edit_text("❌ Git not found on system.")
            return

        try:
            proc = await asyncio.create_subprocess_exec(
                git_path,
                "pull",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
            )
            stdout, _ = await proc.communicate()
            output = stdout.decode().strip()

            if proc.returncode != 0:
                if "Permission denied" in output or "Authentication failed" in output:
                    await msg.edit_text(
                        "❌ Update failed: Private repo access denied. Please check your credentials or use SSH."
                    )
                else:
                    await msg.edit_text(f"⚠️ Git pull failed:\n<pre>{output}</pre>")
                return

            if "Already up to date." in output:
                await msg.edit_text("✅ Bot is already up to date.")
                return

            if len(output) > 4096:
                filename = f"database/{uuid.uuid4().hex}.txt"
                os.makedirs(os.path.dirname(filename), exist_ok=True)
                with open(filename, "w", encoding="utf-8") as f:
                    f.write(output)

                await msg.reply_document(
                    document=types.InputFileLocal(filename),
                    caption="<b>Update log:</b>",
                    parse_mode="html",
                    disable_notification=True,
                )
                os.remove(filename)
            else:
                await msg.edit_text(
                    f"✅ Bot updated successfully. Restarting...\n<b>Update Output:</b>\n<pre>{output}</pre>"
                )

        except Exception as e:
            LOGGER.error("Unexpected update error: %s", e)
            await msg.edit_text(f"⚠️ Update error: {e}")
            return

    if active_vc := chat_cache.get_active_chats():
        for chat_id in active_vc:
            await call.end(chat_id)
            await c.sendTextMessage(
                chat_id,
                "🔧 <b>Bot Maintenance</b>\n\n"
                "The bot is being updated/restarted to bring you new features and improvements.\n"
                "Your music playback has been stopped temporarily. Please start again after a minute.\n\n"
                "Thank you for your patience!",
                parse_mode="html",
            )
            await asyncio.sleep(0.5)

    await msg.edit_text("♻️ Restarting the bot...")

    if is_docker():
        await msg.edit_text(
            "🚢 Detected Docker — exiting process to let Docker restart it."
        )
        sys.exit(0)
    else:
        tgmusic_path = shutil.which("start")
        if not tgmusic_path:
            await msg.edit_text("❌ Unable to find 'start' in PATH.")
            return
        execvp("start", ["start"])
