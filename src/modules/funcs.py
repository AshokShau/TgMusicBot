#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re
from typing import Union

from pytdbot import Client, types

from src.helpers import call, db, get_string
from src.helpers import chat_cache
from src.modules.utils import Filter, sec_to_min, is_channel_cmd
from src.modules.utils.admins import is_admin
from src.modules.utils.play_helpers import del_msg, extract_argument


async def is_admin_or_reply(msg: types.Message) -> Union[int, types.Message, types.Error]:
    """
    Check if user is admin and if a song is playing.
    """
    chat_id = msg.chat_id
    _chat_id = await db.get_channel_id(chat_id) if is_channel_cmd(msg.text) else chat_id
    lang = await db.get_lang(chat_id)
    if not chat_cache.is_active(_chat_id):
        return await msg.reply_text(text=get_string("no_song_playing", lang))

    if not await is_admin(chat_id, msg.from_id):
        return await msg.reply_text(text=get_string("admin_required", lang))

    return _chat_id


async def handle_playback_action(
    c: Client, msg: types.Message, action, success_msg: str, fail_msg: str
) -> None:
    """
    Handle playback actions like stop, pause, resume, mute, unmute.
    """
    lang = await db.get_lang(msg.chat_id)
    _chat_id = await is_admin_or_reply(msg)
    if isinstance(_chat_id, types.Error):
        c.logger.warning(f"Error sending reply: {_chat_id}")
        return

    if isinstance(_chat_id, types.Message):
        return

    done = await action(_chat_id)
    if isinstance(done, types.Error):
        await msg.reply_text(f"⚠️ {fail_msg}\n\n{done.message}")
        return

    await msg.reply_text(
        f"{success_msg}\n│ \n{get_string('requested_by', lang)}: {await msg.mention()} 🥀"
    )
    return


@Client.on_message(filters=Filter.command(["playtype", "setPlayType", "cplaytype", "csetPlayType"]))
async def set_play_type(_: Client, msg: types.Message) -> None:
    """
    Set the play type for a given chat.

    The play type is a preference for a given chat that determines how the
    bot should handle song requests. If the preference is 0, the bot will
    immediately play the first search result. If the preference is 1, the
    bot will send a list of songs to choose from.

    Returns
    -------
    None
    """
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return

    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return

    play_type = extract_argument(msg.text, enforce_digit=True)
    if not play_type:
        await msg.reply_text(get_string("set_play_type_usage", lang))
        return

    play_type = int(play_type)
    if play_type not in (0, 1):
        await msg.reply_text(get_string("invalid_play_type", lang))
        return

    await db.set_play_type(chat_id, play_type)
    await msg.reply_text(get_string("play_type_set", lang).format(play_type))


@Client.on_message(filters=Filter.command(["queue", "cqueue"]))
async def queue_info(_: Client, msg: types.Message) -> None:
    """
    Display information about the current queue.
    """
    if msg.chat_id > 0:
        return

    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    _queue = chat_cache.get_queue(chat_id)
    if not _queue:
        await msg.reply_text(text="🛑 The queue is empty. No tracks left to play!")
        return

    if not chat_cache.is_active(chat_id):
        await msg.reply_text(text="❌ No song is currently playing in this chat!")
        return

    chat: types.Chat = await msg.getChat()
    current_song = _queue[0]
    text = (
        f"<b>🎶 Current Queue in {chat.title}:</b>\n\n"
        f"<b>Currently Playing:</b>\n"
        f"‣ <b>{current_song.name[:30]}</b>\n"
        f"   ├ <b>By:</b> {current_song.user}\n"
        f"   ├ <b>Duration:</b> {sec_to_min(current_song.duration)} minutes\n"
        f"   ├ <b>Loop:</b> {current_song.loop}\n"
        f"   └ <b>Played Time:</b> {sec_to_min(await call.played_time(chat.id))} min"
    )

    if queue_remaining := _queue[1:]:
        text += "\n<b>⏭ Next in Queue:</b>\n"
        for i, song in enumerate(queue_remaining, start=1):
            text += (
                f"{i}. <b>{song.name[:30]}</b>\n"
                f"   ├ <b>Duration:</b> {sec_to_min(song.duration)} min\n"
            )

    text += f"\n<b>» Total of {len(_queue)} track(s) in the queue.</b>"
    if len(text) > 4096:
        short_text = f"<b>🎶 Current Queue in {chat.title}:</b>\n\n"
        short_text += "<b>Currently Playing:</b>\n"
        short_text += f"‣ <b>{current_song.name[:30]}</b>\n"
        short_text += f"   ├ <b>By:</b> {current_song.user}\n"
        short_text += (
            f"   ├ <b>Duration:</b> {sec_to_min(current_song.duration)} minutes\n"
        )
        short_text += f"   ├ <b>Loop:</b> {current_song.loop}\n"
        short_text += f"   └ <b>Played Time:</b> {sec_to_min(await call.played_time(chat.id))} min"
        short_text += f"\n\n<b>» Total of {
        len(_queue)} track(s) in the queue.</b>"
        text = short_text
    await msg.reply_text(text, disable_web_page_preview=True)


@Client.on_message(filters=Filter.command(["loop", "cloop"]))
async def modify_loop(c: Client, msg: types.Message) -> None:
    """
    Modify the loop count for the current song.
    """
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return None

    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return None

    if not chat_cache.is_active(chat_id):
        await msg.reply_text(get_string("no_song_playing", lang))
        return None

    args = extract_argument(msg.text, enforce_digit=True)
    if not args:
        await msg.reply_text(get_string("loop_usage", lang))
        return None

    loop = int(args)
    chat_cache.set_loop_count(chat_id, loop)
    action = get_string("loop_disabled" if loop == 0 else "loop_changed", lang).format(
        loop
    )
    reply = await msg.reply_text(
        get_string("loop_reply", lang).format(action, await msg.mention())
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply.message}")
    return None


@Client.on_message(filters=Filter.command(["seek", "cseek"]))
async def seek_song(c: Client, msg: types.Message) -> None:
    """
    Seek to a specific time in the current song.
    """
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return

    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return

    args = extract_argument(msg.text, enforce_digit=True)
    if not args:
        await msg.reply_text(get_string("seek_usage", lang))
        return

    seek_time = int(args)
    if seek_time < 20:
        await msg.reply_text(get_string("seek_invalid", lang))
        return

    curr_song = chat_cache.get_current_song(chat_id)
    if not curr_song:
        await msg.reply_text(get_string("no_song_playing", lang))
        return

    curr_dur = await call.played_time(chat_id)
    if isinstance(curr_dur, types.Error):
        await msg.reply_text(curr_dur.message)
        return

    seek_to = curr_dur + seek_time
    if seek_to >= curr_song.duration:
        await msg.reply_text(
            get_string("seek_error_duration", lang).format(
                sec_to_min(curr_song.duration)
            )
        )
        return

    _seek = await call.seek_stream(
        chat_id,
        curr_song.file_path,
        seek_to,
        curr_song.duration,
        curr_song.is_video,
    )
    if isinstance(_seek, types.Error):
        await msg.reply_text(_seek.message)
        return

    await msg.reply_text(
        get_string("seek_success", lang).format(seek_to, await msg.mention())
    )
    return


def extract_number(text: str) -> float | None:
    match = re.search(r"[-+]?\d*\.?\d+", text)
    return float(match.group()) if match else None


@Client.on_message(filters=Filter.command(["speed", "cspeed"]))
async def change_speed(_: Client, msg: types.Message) -> None:
    """
    Change the playback speed of the current song.
    """
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return

    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return

    args = extract_number(msg.text)
    if args is None:
        await msg.reply_text(get_string("speed_usage", lang))
        return

    if not chat_cache.is_active(chat_id):
        await msg.reply_text(get_string("no_song_playing", lang))
        return

    speed = round(float(args), 2)
    _change_speed = await call.speed_change(chat_id, speed)
    if isinstance(_change_speed, types.Error):
        await msg.reply_text(_change_speed.message)
        return

    await msg.reply_text(
        get_string("speed_changed", lang).format(speed, await msg.mention())
    )
    return


@Client.on_message(filters=Filter.command(["remove", "cremove"]))
async def remove_song(c: Client, msg: types.Message) -> None:
    """Remove a track from the queue."""
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return None

    args = extract_argument(msg.text, enforce_digit=True)
    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return None

    if not chat_cache.is_active(chat_id):
        await msg.reply_text(get_string("no_song_playing", lang))
        return None

    if not args:
        await msg.reply_text(get_string("remove_usage", lang))
        return None

    track_num = int(args)
    _queue = chat_cache.get_queue(chat_id)

    if not _queue:
        await msg.reply_text(get_string("empty_queue", lang))
        return None

    if track_num <= 0 or track_num > len(_queue):
        await msg.reply_text(
            get_string("invalid_track_number", lang).format(len(_queue))
        )
        return None

    chat_cache.remove_track(chat_id, track_num)
    reply = await msg.reply_text(
        get_string("track_removed", lang).format(await msg.mention())
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply}")
    return None


@Client.on_message(filters=Filter.command(["clear", "cclear"]))
async def clear_queue(c: Client, msg: types.Message) -> None:
    """
    Clear the queue.
    """
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel_cmd(msg.text) else msg.chat_id
    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return None

    if not await is_admin(chat_id, msg.from_id):
        await msg.reply_text(get_string("admin_required", lang))
        return None

    if not chat_cache.is_active(chat_id):
        await msg.reply_text(get_string("no_song_playing", lang))
        return None

    if not chat_cache.get_queue(chat_id):
        await msg.reply_text(get_string("empty_queue", lang))
        return None

    chat_cache.clear_chat(chat_id)
    reply = await msg.reply_text(
        get_string("queue_cleared", lang).format(await msg.mention())
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply}")
    return None


@Client.on_message(filters=Filter.command(["stop", "end", "cstop", "cend"]))
async def stop_song(c: Client, msg: types.Message) -> None:
    """
    Stop the current song.
    """
    chat_id = await is_admin_or_reply(msg)
    if isinstance(chat_id, types.Error):
        c.logger.warning(f"Error sending reply: {chat_id}")
        return None

    if isinstance(chat_id, types.Message):
        return None

    _end = await call.end(chat_id)
    if isinstance(_end, types.Error):
        await msg.reply_text(_end.message)
        return None

    lang = await db.get_lang(msg.chat_id)
    await msg.reply_text(get_string("stream_ended", lang).format(await msg.mention()))
    return None


@Client.on_message(filters=Filter.command(["pause", "cpause"]))
async def pause_song(c: Client, msg: types.Message) -> None:
    """Pause the current song."""
    lang = await db.get_lang(msg.chat_id)
    await handle_playback_action(
        c,
        msg,
        call.pause,
        get_string("stream_paused", lang),
        get_string("pause_error", lang),
    )


@Client.on_message(filters=Filter.command(["resume", "cresume"]))
async def resume(c: Client, msg: types.Message) -> None:
    """Resume the current song."""
    lang = await db.get_lang(msg.chat_id)
    await handle_playback_action(
        c,
        msg,
        call.resume,
        get_string("stream_resumed", lang),
        get_string("resume_error", lang),
    )


@Client.on_message(filters=Filter.command(["mute", "cmute"]))
async def mute_song(c: Client, msg: types.Message) -> None:
    """Mute the current song."""
    lang = await db.get_lang(msg.chat_id)
    await handle_playback_action(
        c,
        msg,
        call.mute,
        get_string("stream_muted", lang),
        get_string("mute_error", lang),
    )


@Client.on_message(filters=Filter.command(["unmute", "cunmute"]))
async def unmute_song(c: Client, msg: types.Message) -> None:
    """Unmute the current song."""
    lang = await db.get_lang(msg.chat_id)
    await handle_playback_action(
        c,
        msg,
        call.unmute,
        get_string("stream_unmuted", lang),
        get_string("unmute_error", lang),
    )


@Client.on_message(filters=Filter.command(["volume", "cvolume"]))
async def volume(c: Client, msg: types.Message) -> None:
    """
    Change the volume of the current song.
    """
    chat_id = await is_admin_or_reply(msg)
    if isinstance(chat_id, types.Error):
        c.logger.warning(f"Error sending reply: {chat_id}")
        return None

    if isinstance(chat_id, types.Message):
        return None

    lang = await db.get_lang(msg.chat_id)
    args = extract_argument(msg.text, enforce_digit=True)
    if not args:
        await msg.reply_text(get_string("volume_usage", lang))
        return None

    vol_int = int(args)
    if vol_int == 0:
        await msg.reply_text(get_string("mute_usage", lang))
        return None

    if not 1 <= vol_int <= 200:
        await msg.reply_text(get_string("volume_range_error", lang))
        return None

    done = await call.change_volume(chat_id, vol_int)
    if isinstance(done, types.Error):
        await msg.reply_text(done.message)
        return None

    await msg.reply_text(
        get_string("volume_set", lang).format(vol_int, await msg.mention())
    )
    return None


@Client.on_message(filters=Filter.command(["skip", "cskip"]))
async def skip_song(c: Client, msg: types.Message) -> None:
    """
    Skip the current song.
    """
    lang = await db.get_lang(msg.chat_id)
    chat_id = await is_admin_or_reply(msg)
    if isinstance(chat_id, types.Error):
        c.logger.warning(f"Error sending reply: {chat_id}")
        return None

    if isinstance(chat_id, types.Message):
        return None

    await del_msg(msg)

    done = await call.play_next(chat_id)
    if isinstance(done, types.Error):
        await msg.reply_text(
            f"⚠️ {get_string('error_occurred', lang)}\n\n{done.message}"
        )
        return None

    reply = await msg.reply_text(
        f"⏭️ {get_string('song_skipped', lang)}\n│ \n└ {get_string('requested_by', lang)}: {await msg.mention()} 🥀"
    )
    if isinstance(reply, types.Error):
        c.logger.warning(f"Error sending reply: {reply}")

    return None
