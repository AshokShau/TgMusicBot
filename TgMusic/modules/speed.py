#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.


import re

from pytdbot import Client, types

from TgMusic.core import Filter, admins_only, call, chat_cache


def extract_number(text: str) -> float | None:
    """Extract a numerical value from text."""
    match = re.search(r"[-+]?\d*\.?\d+", text)
    return float(match.group()) if match else None


@Client.on_message(filters=Filter.command("speed"))
@admins_only(is_bot=True, is_auth=True)
async def change_speed(_: Client, msg: types.Message) -> None:
    """Adjust the playback speed of the current track."""
    chat_id = msg.chat_id
    if chat_id > 0:
        return

    args = extract_number(msg.text)
    if args is None:
        await msg.reply_text(
            "ℹ️ <b>Usage:</b> <code>/speed [value]</code>\n"
            "Example: <code>/speed 1.5</code> for 1.5x speed\n"
            "Range: 0.5x to 4.0x"
        )
        return

    if not chat_cache.is_active(chat_id):
        await msg.reply_text("⏸ No track is currently playing.")
        return

    speed = round(float(args), 2)
    if speed < 0.5 or speed > 4.0:
        await msg.reply_text("⚠️ Speed must be between 0.5x and 4.0x")
        return

    _change_speed = await call.speed_change(chat_id, speed)
    if isinstance(_change_speed, types.Error):
        await msg.reply_text(f"⚠️ <b>Error:</b> {_change_speed.message}")
        return

    await msg.reply_text(
        f"🎚️ Playback speed set to <b>{speed}x</b> by {await msg.mention()}"
    )
