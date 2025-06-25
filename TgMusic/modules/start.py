#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pytdbot import Client, types

from TgMusic import __version__
from TgMusic.core import (
    config,
    Filter,
    SupportButton,
)
from TgMusic.core.buttons import add_me_markup, HelpMenu, BackHelpMenu


@Client.on_message(filters=Filter.command(["start", "help"]))
async def start_cmd(c: Client, message: types.Message):
    chat_id = message.chat_id
    bot_name = c.me.first_name
    mention = await message.mention()

    if chat_id < 0:  # Group chat
        welcome_text = (
            f"🎵 <b>Hello {mention}!</b>\n\n"
            f"<b>{bot_name}</b> is now active in this group.\n"
            "Here’s what I can do:\n"
            "• High-quality music streaming\n"
            "• Supports YouTube, Spotify, and more\n"
            "• Powerful controls for seamless playback\n\n"
            f"💬 <a href='{config.SUPPORT_GROUP}'>Need help? Join our Support Chat</a>"
        )
        await message.reply_text(
            text=welcome_text,
            disable_web_page_preview=True,
            reply_markup=SupportButton,
        )

    else:  # Private chat
        welcome_text = (
            f"✨ <b>Hey {mention}!</b>\n\n"
            f"You’re chatting with <b>{bot_name}</b> — your advanced music assistant.\n"
            f"<code>Version: v{__version__}</code>\n\n"
            "🎧 <b>What I Offer:</b>\n"
            "• Crystal-clear audio playback\n"
            "• Instant song search across platforms\n"
            "• Playlist support & 24/7 uptime\n\n"
            "🔽 <i>Tap the button below to get started!</i>"
        )
        bot_username = c.me.usernames.editable_username
        await message.reply_text(
            text=welcome_text,
            reply_markup=add_me_markup(bot_username),
        )


@Client.on_updateNewCallbackQuery(filters=Filter.regex(r"help_\w+"))
async def callback_query_help(c: Client, message: types.UpdateNewCallbackQuery) -> None:
    data = message.payload.data.decode()

    if data == "help_all":
        user = await c.getUser(message.sender_user_id)

        await message.answer("📚 Opening Help Menu...")
        welcome_text = (
            f"👋 <b>Hello {user.first_name}!</b>\n\n"
            f"Welcome to <b>{c.me.first_name}</b> — your ultimate music bot.\n"
            f"<code>Version: v{__version__}</code>\n\n"
            "💡 <b>What makes me special?</b>\n"
            "• YouTube, Spotify, Apple Music, SoundCloud support\n"
            "• Advanced queue and playback controls\n"
            "• Private and group usage\n\n"
            "🔍 <i>Select a help category below to continue.</i>"
        )
        await message.edit_message_text(text=welcome_text, reply_markup=HelpMenu)
        return

    help_categories = {
        "help_user": {
            "title": "🎧 User Commands",
            "content": (
                "<b>▶️ Playback:</b>\n"
                "• <code>/play [song]</code> — Play audio in VC\n"
                "• <code>/vplay [video]</code> — Play video in VC\n"
                "• <code>/search</code> — Browse tracks before playing\n"
                "• <code>/lyrics</code> — Get lyrics for a song\n\n"
                "<b>🛠 Utilities:</b>\n"
                "• <code>/start</code> — Intro message\n"
                "• <code>/privacy</code> — Privacy policy\n"
                "• <code>/lang</code> — Change language"
            ),
            "markup": BackHelpMenu,
        },
        "help_admin": {
            "title": "⚙️ Admin Commands",
            "content": (
                "<b>🎛 Playback Controls:</b>\n"
                "• <code>/skip</code> — Skip current track\n"
                "• <code>/pause</code> — Pause playback\n"
                "• <code>/resume</code> — Resume playback\n"
                "• <code>/seek [sec]</code> — Jump to a position\n"
                "• <code>/volume [1-200]</code> — Set playback volume\n\n"
                "<b>📋 Queue Management:</b>\n"
                "• <code>/queue</code> — View track queue\n"
                "• <code>/remove [x]</code> — Remove track number x\n"
                "• <code>/clear</code> — Clear the entire queue\n"
                "• <code>/loop [0-10]</code> — Repeat queue x times"
            ),
            "markup": BackHelpMenu,
        },
        "help_owner": {
            "title": "🔐 Owner Commands",
            "content": (
                "<b>👑 Permissions:</b>\n"
                "• <code>/auth [reply]</code> — Grant admin access\n"
                "• <code>/unauth [reply]</code> — Revoke admin access\n"
                "• <code>/authlist</code> — View authorized users\n\n"
                "<b>⚙️ Settings:</b>\n"
                "• <code>/buttons</code> — Toggle control buttons\n"
                "• <code>/thumb</code> — Toggle thumbnail mode"
            ),
            "markup": BackHelpMenu,
        },
        "help_devs": {
            "title": "🛠 Developer Tools",
            "content": (
                "<b>📊 System Tools:</b>\n"
                "• <code>/stats</code> — Show usage stats\n"
                "• <code>/logger</code> — Toggle log mode\n"
                "• <code>/broadcast</code> — Send a message to all\n\n"
                "<b>🧹 Maintenance:</b>\n"
                "• <code>/activevc</code> — Show active voice chats\n"
                "• <code>/clearallassistants</code> — Remove all assistants\n"
                "• <code>/autoend</code> — Enable auto-leave when VC is empty"
            ),
            "markup": BackHelpMenu,
        },
    }

    if category := help_categories.get(data):
        await message.answer(f"📖 {category['title']}")
        formatted_text = (
            f"<b>{category['title']}</b>\n\n"
            f"{category['content']}\n\n"
            "🔙 <i>Use the buttons below to go back.</i>"
        )
        await message.edit_message_text(
            text=formatted_text, reply_markup=category["markup"]
        )
        return

    await message.answer("⚠️ Unknown command category.")
