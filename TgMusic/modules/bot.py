import time
from datetime import datetime

from cachetools import TTLCache
from pytdbot import Client, types

from TgMusic import StartTime
from TgMusic.core import (
    chat_invite_cache,
    user_status_cache,
    chat_cache,
    call,
    Filter,
)
from TgMusic.core.admins import load_admin_cache
from TgMusic.modules.utils import sec_to_min


@Client.on_message(filters=Filter.command("privacy"))
async def privacy_handler(c: Client, message: types.Message):
    """
    Handle the /privacy command to display privacy policy.
    """
    bot_name = c.me.first_name
    text = f"""
    <u><b>Privacy Policy for {bot_name}:</b></u>

<b>1. Data Storage:</b>
- {bot_name} does not store any personal data on the user's device.
- We do not collect or store any data about your device or personal browsing activity.

<b>2. What We Collect:</b>
- We only collect your Telegram <b>user ID</b> and <b>chat ID</b> to provide the music streaming and interaction functionalities of the bot.
- No personal data such as your name, phone number, or location is collected.

<b>3. Data Usage:</b>
- The collected data (Telegram UserID, ChatID) is used strictly to provide the music streaming and interaction functionalities of the bot.
- We do not use this data for any marketing or commercial purposes.

<b>4. Data Sharing:</b>
- We do not share any of your personal or chat data with any third parties, organizations, or individuals.
- No sensitive data is sold, rented, or traded to any outside entities.

<b>5. Data Security:</b>
- We take reasonable security measures to protect the data we collect. This includes standard practices like encryption and safe storage.
- However, we cannot guarantee the absolute security of your data, as no online service is 100% secure.

<b>6. Cookies and Tracking:</b>
- {bot_name} does not use cookies or similar tracking technologies to collect personal information or track your behavior.

<b>7. Third-Party Services:</b>
- {bot_name} does not integrate with any third-party services that collect or process your personal information, aside from Telegram's own infrastructure.

<b>8. Your Rights:</b>
- You have the right to request the deletion of your data. Since we only store your Telegram ID and chat ID temporarily to function properly, these can be removed upon request.
- You may also revoke access to the bot at any time by removing or blocking it from your chats.

<b>9. Changes to the Privacy Policy:</b>
- We may update this privacy policy from time to time. Any changes will be communicated through updates within the bot.

<b>10. Contact Us:</b>
If you have any questions or concerns about our privacy policy, feel free to contact us at <a href="https://t.me/GuardxSupport">Support Group</a>

──────────────────
<b>Note:</b> This privacy policy is in place to help you understand how your data is handled and to ensure that your experience with {bot_name} is safe and respectful.
    """

    reply = await message.reply_text(text)
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending privacy policy message:{reply.message}")
    return


rate_limit_cache = TTLCache(maxsize=100, ttl=180)


@Client.on_message(filters=Filter.command(["reload"]))
async def reload_cmd(c: Client, message: types.Message) -> None:
    """Handle the /reload command to reload the bot."""
    user_id = message.from_id
    chat_id = message.chat_id
    if chat_id > 0:
        reply = await message.reply_text(
            "🚫 This command can only be used in SuperGroups only."
        )
        if isinstance(reply, types.Error):
            c.logger.warning(f"Error sending message: {reply} for chat {chat_id}")
        return None

    if user_id in rate_limit_cache:
        last_used_time = rate_limit_cache[user_id]
        time_remaining = 180 - (datetime.now() - last_used_time).total_seconds()
        reply = await message.reply_text(
            f"🚫 You can use this command again in ({sec_to_min(time_remaining)} Min)"
        )
        if isinstance(reply, types.Error):
            c.logger.warning(f"Error sending message: {reply} for chat {chat_id}")
        return None

    rate_limit_cache[user_id] = datetime.now()
    reply = await message.reply_text("🔄 Reloading...")
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending message: {reply} for chat {chat_id}")
        return None

    ub = await call.get_client(chat_id)
    if isinstance(ub, types.Error):
        await reply.edit_text(ub.message)
        return None

    chat_invite_cache.pop(chat_id, None)
    user_key = f"{chat_id}:{ub.me.id}"
    user_status_cache.pop(user_key, None)

    if not chat_cache.is_active(chat_id):
        chat_cache.clear_chat(chat_id)

    load_admins, _ = await load_admin_cache(c, chat_id, True)
    ub_stats = await call.check_user_status(chat_id)
    if isinstance(ub_stats, types.Error):
        ub_stats = ub_stats.message

    loaded = "✅" if load_admins else "❌"
    text = (
        f"<b>Assistant Status:</b> {ub_stats.getType()}\n"
        f"<b>Admins Loaded:</b> {loaded}\n"
        f"<b>» Reloaded by:</b> {await message.mention()}"
    )

    reply = await reply.edit_text(text)
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending message: {reply} for chat {chat_id}")
    return None


@Client.on_message(filters=Filter.command("ping"))
async def ping_cmd(client: Client, message: types.Message) -> None:
    """
    Handle the /ping command to check bot performance metrics.
    """
    response = await call.stats_call(message.chat_id if message.chat_id < 0 else 1)
    if isinstance(response, types.Error):
        call_ping = response.message
        cpu_usage = "Unavailable"
    else:
        call_ping, cpu_usage = response
    call_ping_info = f"{call_ping:.2f} ms"
    cpu_info = f"{cpu_usage:.2f}%"
    uptime = datetime.now() - StartTime
    uptime_str = str(uptime).split(".")[0]
    start_time = time.monotonic()
    reply_msg = await message.reply_text("🏓 Pinging...")
    latency = (time.monotonic() - start_time) * 1000  # ms
    response = (
        "📊 <b>System Performance Metrics</b>\n\n"
        f"⏱️ <b>Bot Latency:</b> <code>{latency:.2f} ms</code>\n"
        f"🕒 <b>Uptime:</b> <code>{uptime_str}</code>\n"
        f"🧠 <b>CPU Usage:</b> <code>{cpu_info}</code>\n"
        f"📞 <b>NTgCalls Ping:</b> <code>{call_ping_info}</code>\n"
    )
    done = await reply_msg.edit_text(response, disable_web_page_preview=True)
    if isinstance(done, types.Error):
        client.logger.warning(f"Error sending message: {done}")
    return None
