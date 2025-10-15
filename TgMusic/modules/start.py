#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic import __version__
from TgMusic.core import (
    Filter,
    SupportButton,
    config,
)
from TgMusic.core.buttons import BackHelpMenu, HelpMenu, add_me_markup

START_TEXT = """
ʜᴇʏ {};

◎ ᴛʜɪꜱ ɪꜱ {}!
➻ ᴀ ꜰᴀꜱᴛ & ᴘᴏᴡᴇʀꜰᴜʟ ᴛᴇʟᴇɢʀᴀᴍ ᴍᴜꜱɪᴄ ᴘʟᴀʏᴇʀ ʙᴏᴛ ᴡɪᴛʜ ꜱᴏᴍᴇ ᴀᴡᴇꜱᴏᴍᴇ ꜰᴇᴀᴛᴜʀᴇꜱ.

ꜱᴜᴘᴘᴏʀᴛᴇᴅ ᴘʟᴀᴛꜰᴏʀᴍꜱ: ʏᴏᴜᴛᴜʙᴇ, ꜱᴘᴏᴛɪꜰʏ, ᴊɪᴏꜱᴀᴀᴠɴ, ᴀᴘᴘʟᴇ ᴍᴜꜱɪᴄ ᴀɴᴅ ꜱᴏᴜɴᴅᴄʟᴏᴜᴅ.

---
◎ ᴄʟɪᴄᴋ ᴏɴ ᴛʜᴇ ʜᴇʟᴘ ʙᴜᴛᴛᴏɴ ᴛᴏ ɢᴇᴛ ɪɴꜰᴏʀᴍᴀᴛɪᴏɴ ᴀʙᴏᴜᴛ ᴍʏ ᴍᴏᴅᴜʟᴇꜱ ᴀɴᴅ ᴄᴏᴍᴍᴀɴᴅꜱ.
"""

HELP_CATEGORIES = {
    "help_user": {
        "title": "🎧 User Commands",
        "content": """
<b>▶️ Playback:</b>
• <code>/play [song]</code> — Play audio in VC
• <code>/vplay [video]</code> — Play video in VC

<b>🛠 Utilities:</b>
• <code>/start</code> — Intro message
• <code>/privacy</code> — Privacy policy
• <code>/queue</code> — View track queue
""",
        "markup": BackHelpMenu,
    },
    "help_admin": {
        "title": "⚙️ Admin Commands",
        "content": """
<b>🎛 Playback Controls:</b>
• <code>/skip</code> — Skip current track
• <code>/pause</code> — Pause playback
• <code>/resume</code> — Resume playback
• <code>/seek [sec]</code> — Jump to a position
• <code>/volume [1-200]</code> — Set playback volume

<b>📋 Queue Management:</b>
• <code>/remove [x]</code> — Remove track number x
• <code>/clear</code> — Clear the entire queue
• <code>/loop [0-10]</code> — Repeat queue x times

<b>👑 Permissions:</b>
• <code>/auth [reply]</code> — Grant approval to use commands 
• <code>/unauth [reply]</code> — Revoke authorization
• <code>/authlist</code> — View authorized users
""",
        "markup": BackHelpMenu,
    },
    "help_owner": {
        "title": "🔐 Owner Commands",
        "content": """
<b>⚙️ Settings:</b>
• <code>/buttons</code> — Toggle control buttons
• <code>/thumb</code> — Toggle thumbnail mode
""",
        "markup": BackHelpMenu,
    },
    "help_devs": {
        "title": "🛠 Developer Tools",
        "content": """
<b>📊 System Tools:</b>
• <code>/stats</code> — Show usage stats
• <code>/logger</code> — Toggle log mode
• <code>/broadcast</code> — Send a message to all

<b>🧹 Maintenance:</b>
• <code>/activevc</code> — Show active voice chats
• <code>/clearallassistants</code> — Remove all assistants data from DB
• <code>/autoend</code> — Enable auto-leave when VC is empty
""",
        "markup": BackHelpMenu,
    },
}


@Client.on_message(filters=Filter.command(["start", "help"]))
async def start_cmd(c: Client, message: types.Message) -> None:
    """Handle /start and /help commands."""
    chat_id = message.chat_id
    bot_name = c.me.first_name
    mention = await message.mention()

    if chat_id < 0:  # Group
        welcome_text = (
            f"🎵 <b>Hello {mention}!</b>\n\n"
            f"<b>{bot_name}</b> is now active in this group.\n"
            "Here’s what I can do:\n"
            "• High-quality music streaming\n"
            "• Supports YouTube, Spotify, and more\n"
            "• Powerful controls for seamless playback\n\n"
            f"💬 <a href='{config.SUPPORT_GROUP}'>Need help? Join our Support Chat</a>"
        )
        reply = await message.reply_text(
            text=welcome_text,
            disable_web_page_preview=True,
            reply_markup=SupportButton,
        )
    else:  # Private chat
        reply = await message.reply_photo(
            photo=config.START_IMG,
            caption=START_TEXT.format(mention, bot_name),
            reply_markup=add_me_markup(c.me.usernames.editable_username),
        )

    if isinstance(reply, types.Error):
        c.logger.warning(f"Failed to send start/help reply: {reply.message}")


@Client.on_updateNewCallbackQuery(filters=Filter.regex(r"help_\w+"))
async def callback_query_help(c: Client, message: types.UpdateNewCallbackQuery) -> None:
    """Handle help menu callback queries."""
    data = message.payload.data.decode()

    if data == "help_all":
        user = await c.getUser(message.sender_user_id)
        await message.answer("📚 Opening Help Menu...")
        text = (
            f"👋 <b>Hello {user.first_name}!</b>\n\n"
            f"Welcome to <b>{c.me.first_name}</b> — your ultimate music bot.\n"
            f"<code>Version: v{__version__}</code>\n\n"
            "💡 <b>What makes me special?</b>\n"
            "• YouTube, Spotify, Apple Music, SoundCloud support\n"
            "• Advanced queue and playback controls\n"
            "• Private and group usage\n\n"
            "🔍 <i>Select a help category below to continue.</i>"
        )

        result = await message.edit_message_caption(text, reply_markup=HelpMenu)
        if isinstance(result, types.Error):
            c.logger.error(f"Edit failed: {result.message}")
        return None

    if data == "help_back":
        await message.answer("🏠 Returning to home...")
        user = await c.getUser(message.sender_user_id)

        result = await message.edit_message_caption(
            START_TEXT.format(user.first_name, c.me.first_name),
            reply_markup=add_me_markup(c.me.usernames.editable_username),
        )
        if isinstance(result, types.Error):
            c.logger.error(f"Edit failed: {result.message}")
        return None

    if category := HELP_CATEGORIES.get(data):
        await message.answer(f"📖 {category['title']}")
        text = f"<b>{category['title']}</b>\n\n{category['content']}\n\n🔙 <i>Use the buttons below to go back.</i>"

        result = await message.edit_message_caption(
            text, reply_markup=category["markup"]
        )
        if isinstance(result, types.Error):
            c.logger.error(f"Edit failed: {result.message}")
        return None

    await message.answer("⚠️ Unknown command category.")
    return None
