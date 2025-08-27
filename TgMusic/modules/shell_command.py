#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import os
import uuid

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only
from TgMusic.logger import LOGGER


async def run_shell_command(cmd: str, timeout: int = 60) -> tuple[str, str, int]:
    """Execute shell command and return stdout, stderr, returncode."""
    process = await asyncio.create_subprocess_shell(
        cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )

    try:
        stdout, stderr = await asyncio.wait_for(process.communicate(), timeout=timeout)
    except asyncio.TimeoutError:
        process.kill()
        await process.wait()
        return "", f"Command timed out after {timeout} seconds", -1

    return stdout.decode().strip(), stderr.decode().strip(), process.returncode


async def shellrunner(message: types.Message) -> types.Ok | types.Error | types.Message:
    text = message.text.split(None, 1)
    if len(text) <= 1:
        reply = await message.reply_text("Usage: /sh &lt cmd &gt")
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)
        return types.Ok()

    command = text[1]
    """
    # Security check - prevent dangerous commands
    if any(blocked in command.lower() for blocked in [
        'rm -rf', 'sudo', 'dd ', 'mkfs', 'fdisk',
        ':(){:|:&};:', 'chmod 777', 'wget', 'curl'
    ]):
        return await message.reply_text("⚠️ Dangerous command blocked!")
    """

    try:
        # Execute single command or multiple commands separated by newlines
        if "\n" in command:
            commands = [cmd.strip() for cmd in command.split("\n") if cmd.strip()]
            output_parts = []

            for cmd in commands:
                stdout, stderr, retcode = await run_shell_command(cmd)

                output_parts.append(f"<b>🚀 Command:</b> <code>{cmd}</code>")
                if stdout:
                    output_parts.append(f"<b>📤 Output:</b>\n<pre>{stdout}</pre>")
                if stderr:
                    output_parts.append(f"<b>❌ Error:</b>\n<pre>{stderr}</pre>")
                output_parts.append(f"<b>🔢 Exit Code:</b> <code>{retcode}</code>\n")

            output = "\n".join(output_parts)
        else:
            stdout, stderr, retcode = await run_shell_command(command)

            output = f"<b>🚀 Command:</b> <code>{command}</code>\n"
            if stdout:
                output += f"<b>📤 Output:</b>\n<pre>{stdout}</pre>\n"
            if stderr:
                output += f"<b>❌ Error:</b>\n<pre>{stderr}</pre>\n"
            output += f"<b>🔢 Exit Code:</b> <code>{retcode}</code>"

        # Handle empty output
        if not output.strip():
            output = "<b>📭 No output was returned</b>"

        if len(output) <= 2000:
            return await message.reply_text(str(output), parse_mode="html")

        filename = f"database/{uuid.uuid4().hex}.txt"
        with open(filename, "w", encoding="utf-8") as file:
            file.write(output)
        reply = await message.reply_document(
            document=types.InputFileLocal(filename),
            caption="📁 Output too large, sending as file:",
            disable_notification=True,
            parse_mode="html",
        )
        if isinstance(reply, types.Error):
            LOGGER.warning(reply.message)

        if os.path.exists(filename):
            os.remove(filename)

        return types.Ok()
    except Exception as e:
        return await message.reply_text(
            f"⚠️ <b>Error:</b>\n<pre>{str(e)}</pre>", parse_mode="html"
        )


@Client.on_message(filters=Filter.command("sh"))
@admins_only(only_dev=True)
async def shell_command(_: Client, m: types.Message) -> None:
    done = await shellrunner(m)
    if isinstance(done, types.Error):
        LOGGER.warning(done.message)
        return None
    return None
