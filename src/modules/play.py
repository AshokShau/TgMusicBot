#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re
from types import NoneType

from pytdbot import Client, types

from src.helpers import (
    CachedTrack,
    MusicServiceWrapper,
    MusicTrack,
    PlatformTracks,
    Telegram,
    YouTubeData,
    call,
    db,
    get_string,
)
from src.helpers import chat_cache, ChannelPlay

from src.logger import LOGGER
from src.modules.utils import Filter, SupportButton, get_audio_duration, sec_to_min, check_user_status, is_channel_cmd, \
    control_buttons
from src.modules.utils import user_status_cache, join_ub
from src.modules.utils.admins import is_admin, load_admin_cache, is_owner

from src.modules.utils.play_helpers import (
    del_msg,
    edit_text,
    extract_argument,
    get_url,
    unban_ub,
)
from src.modules.utils.thumbnails import gen_thumb


def _get_jiosaavn_url(track_id: str) -> str:
    """
    Generate JioSaavn URL from track ID.
    """
    try:
        title, song_id = track_id.rsplit("/", 1)
    except ValueError:
        return ""
    title = re.sub(r'[\(\)"\',]', "", title.lower()).replace(" ", "-")
    return f"https://www.jiosaavn.com/song/{title}/{song_id}"


def _get_platform_url(platform: str, track_id: str) -> str:
    """
    Generate platform URL from track ID.
    """
    platform = platform.lower()
    if not track_id:
        return ""
    url_map = {
        "youtube": f"https://youtube.com/watch?v={track_id}",
        "spotify": f"https://open.spotify.com/track/{track_id}",
        "jiosaavn": _get_jiosaavn_url(track_id),
    }
    return url_map.get(platform, "")


def build_song_selection_message(
    user_by: str, tracks: list[MusicTrack]
) -> tuple[str, types.ReplyMarkupInlineKeyboard]:
    """
    Build a message and inline keyboard for song selection.
    """
    text = f"{user_by}, select a song to play:" if user_by else "Select a song to play:"
    buttons = [
        [
            types.InlineKeyboardButton(
                f"{rec.name[:18]} - {rec.artist}",
                type=types.InlineKeyboardButtonTypeCallback(
                    f"play_{rec.platform.lower()}_{rec.id}".encode()
                ),
            )
        ]
        for rec in tracks[:4]
    ]
    return text, types.ReplyMarkupInlineKeyboard(buttons)


async def _update_msg_with_thumb(
    c: Client,
    msg: types.Message,
    text: str,
    thumb: str,
    button: types.ReplyMarkupInlineKeyboard,
):
    """
    Update a message with thumbnail if available.
    """
    if not thumb:
        return await edit_text(msg, text=text, reply_markup=button)

    parsed_text = await c.parseTextEntities(text, types.TextParseModeHTML())
    if isinstance(parsed_text, types.Error):
        return await edit_text(msg, text=parsed_text.message, reply_markup=button)

    input_content = types.InputMessagePhoto(
        types.InputFileLocal(thumb), caption=parsed_text
    )
    reply = await c.editMessageMedia(
        chat_id=msg.chat_id,
        message_id=msg.id,
        input_message_content=input_content,
        reply_markup=button,
    )

    return (
        await edit_text(msg, text=str(reply), reply_markup=button)
        if isinstance(reply, types.Error)
        else reply
    )


async def _handle_single_track(
    c: Client,
    msg: types.Message,
    channel: ChannelPlay,
    track: MusicTrack,
    user_by: str,
    file_path: str = None,
    is_video: bool = False,
):
    """
    Handle playback of a single track.
    """
    chat_id = channel.chat_id
    lang = await db.get_lang(msg.chat_id)
    song = CachedTrack(
        name=track.name,
        artist=track.artist,
        track_id=track.id,
        loop=0,
        duration=track.duration,
        file_path=file_path or "",
        thumbnail=track.cover,
        user=user_by,
        platform=track.platform,
        is_video=is_video,
        url=track.url,
        channel=channel,
    )

    if not song.file_path:
        if file_path := await call.song_download(song):
            song.file_path = file_path
        else:
            return await edit_text(
                msg, f"❌ {get_string('error_downloading_song', lang)}"
            )

    song.duration = song.duration or await get_audio_duration(song.file_path)
    if chat_cache.is_active(chat_id):
        queue = chat_cache.get_queue(chat_id)
        chat_cache.add_song(chat_id, song)
        text = (
            f"<b>➻ {get_string('added_to_queue_at', lang)} #{len(queue)}:</b>\n\n"
            f"‣ <b>{get_string('title', lang)}:</b> {song.name}\n"
            f"‣ <b>{get_string('duration', lang)}:</b> {sec_to_min(song.duration)}\n"
            f"‣ <b>{get_string('requested_by', lang)}:</b> {song.user}"
        )
        thumb = await gen_thumb(song) if await db.get_thumb_status(chat_id) else ""
        await _update_msg_with_thumb(
            c,
            msg,
            text,
            thumb,
            control_buttons("play", song.channel.is_channel) if await db.get_buttons_status(chat_id) else None,
        )
        return None

    chat_cache.set_active(chat_id, True)
    chat_cache.add_song(chat_id, song)

    _call = await call.play_media(chat_id, song.file_path, video=is_video)
    if isinstance(_call, types.Error):
        return await edit_text(msg, text=f"⚠️ {str(_call)}")

    thumb = await gen_thumb(song) if await db.get_thumb_status(chat_id) else ""
    text = (
        f"🎵 <b>{get_string('now_playing', lang)}:</b>\n\n"
        f"‣ <b>{get_string('title', lang)}:</b> {song.name}\n"
        f"‣ <b>{get_string('duration', lang)}:</b> {sec_to_min(song.duration)}\n"
        f"‣ <b>{get_string('requested_by', lang)}:</b> {song.user}"
    )

    reply = await _update_msg_with_thumb(
        c,
        msg,
        text,
        thumb,
        control_buttons("play", channel.is_channel) if await db.get_buttons_status(chat_id) else None,
    )
    if isinstance(reply, types.Error):
        LOGGER.info("sending reply: %s", reply)
        return None
    return None


async def _handle_multiple_tracks(msg: types.Message, tracks: list[MusicTrack], user_by: str, channel: ChannelPlay):
    """
    Handle multiple tracks (playlist/album).
    """
    chat_id = channel.chat_id
    lang = await db.get_lang(msg.chat_id)
    is_active = chat_cache.is_active(chat_id)
    queue = chat_cache.get_queue(chat_id)
    text = (
        "<b>➻ "
        + get_string("added_to_queue", lang)
        + ":</b>\n<blockquote expandable>\n"
    )


    for index, track in enumerate(tracks):
        position = len(queue) + index
        chat_cache.add_song(
            chat_id,
            CachedTrack(
                name=track.name,
                artist=track.artist,
                track_id=track.id,
                loop=1 if not is_active and index == 0 else 0,
                duration=track.duration,
                thumbnail=track.cover,
                user=user_by,
                file_path="",
                platform=track.platform,
                is_video=False,
                url=track.url,
                channel=channel,
            ),
        )
        text += f"<b>{position}.</b> {track.name}\n└ {get_string('duration', lang)}: {sec_to_min(track.duration)}\n"

    text += "</blockquote>\n"
    total_dur = sum(t.duration for t in tracks)
    text += (
        f"<b>📋 {get_string('total_queue', lang)}:</b> {len(chat_cache.get_queue(chat_id))}\n"
        f"<b>⏱️ {get_string('total_duration', lang)}:</b> {sec_to_min(total_dur)}\n"
        f"<b>👤 {get_string('requested_by', lang)}:</b> {user_by}"
    )

    if len(text) > 4096:
        text = (
            f"<b>📋 {get_string('total_queue', lang)}:</b> {len(chat_cache.get_queue(chat_id))}\n"
            f"<b>⏱️ {get_string('total_duration', lang)}:</b> {sec_to_min(total_dur)}\n"
            f"<b>👤 {get_string('requested_by', lang)}:</b> {user_by}"
        )

    if not is_active:
        await call.play_next(chat_id)

    await edit_text(msg, text, reply_markup=control_buttons("play", channel.is_channel))


async def play_music(
    c: Client,
    msg: types.Message,
    url_data: PlatformTracks,
    user_by: str,
    channel: ChannelPlay,
    tg_file_path: str = None,
    is_video: bool = False,
):
    """
    Handle playing music from a given URL or file.
    """
    lang = await db.get_lang(msg.chat_id)
    if not url_data or not url_data.tracks:
        return await edit_text(msg, get_string("unable_to_retrieve_song_info", lang))

    await edit_text(msg, text=get_string("song_found_downloading", lang))
    if len(url_data.tracks) == 1:
        return await _handle_single_track(
            c, msg, channel, url_data.tracks[0], user_by, tg_file_path, is_video
        )
    return await _handle_multiple_tracks(msg, url_data.tracks, user_by, channel)


async def _handle_recommendations(
    _: Client, msg: types.Message, wrapper: MusicServiceWrapper
):
    """
    Show music recommendations when no query is provided.
    """
    lang = await db.get_lang(msg.chat_id)
    recommendations = await wrapper.get_recommendations()

    text = get_string("usage_play_song_name", lang)
    if not recommendations:
        await edit_text(msg, text=text, reply_markup=SupportButton)
        return

    text, keyboard = build_song_selection_message("", recommendations.tracks)
    await edit_text(
        msg, text=text, reply_markup=keyboard, disable_web_page_preview=True
    )


async def _handle_telegram_file(
    c: Client,
    channel: ChannelPlay,
    reply: types.Message,
    reply_message: types.Message,
    user_by: str,
):
    """
    Handle Telegram audio/video files.
    """
    lang = await db.get_lang(channel.chat_id)
    telegram = Telegram(reply)

    # Determine if the message contains a video (Document or Video type)
    content = reply.content
    docs_vid = (
        isinstance(content, types.Document) and content.mime_type.startswith("video/")
    ) or (
        isinstance(content, types.MessageDocument)
        and content.document.mime_type.startswith("video/")
    )
    is_video = isinstance(content, types.MessageVideo) or docs_vid

    # Download the file
    file_path, file_name = await telegram.dl(reply_message)
    if isinstance(file_path, types.Error):
        return await edit_text(
            reply_message,
            text=get_string("telegram_file_download_failed", lang).format(
                file=file_name, error=file_path.message
            ),
        )

    duration = await get_audio_duration(file_path.path)

    # Wrap in a consistent track structure
    _song = PlatformTracks(
        tracks=[
            MusicTrack(
                name=file_name,
                artist="AshokShau",
                id=reply.remote_unique_file_id,
                year=0,
                cover="",
                duration=duration,
                url="",
                platform="telegram",
            )
        ]
    )

    await play_music(c, reply_message, _song, user_by, channel, file_path.path, is_video)
    return None


async def _handle_text_search(
    c: Client,
    msg: types.Message,
    channel: ChannelPlay,
    wrapper: MusicServiceWrapper,
    user_by: str,
):
    """
    Handle text-based music search.
    """
    chat_id = channel.chat_id
    lang = await db.get_lang(chat_id)
    play_type = await db.get_play_type(chat_id)
    search = await wrapper.search()

    if not search or not search.tracks:
        return await edit_text(
            msg,
            text=get_string("no_results_bug", lang),
            reply_markup=SupportButton,
        )

    if play_type == 0:
        url = search.tracks[0].url
        if song := await MusicServiceWrapper(url).get_info():
            return await play_music(c, msg, song, user_by, channel)

        return await edit_text(
            msg,
            text=get_string("failed_song_info", lang),
            reply_markup=SupportButton,
        )

    text, keyboard = build_song_selection_message(user_by, search.tracks)
    await edit_text(
        msg, text=text, reply_markup=keyboard, disable_web_page_preview=True
    )
    return None


async def handle_play_command(c: Client, msg: types.Message, is_video: bool = False):
    """
    Generic handler for /play and /vplay.
    """
    is_channel = is_channel_cmd(msg.text)
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel else msg.chat_id
    channel = ChannelPlay(
        chat_id=chat_id,
        is_channel=is_channel and chat_id != msg.chat_id,
    )

    lang = await db.get_lang(chat_id)
    if chat_id > 0:
        return await msg.reply_text(get_string("only_supergroup", lang))

    # Queue limit
    queue = chat_cache.get_queue(chat_id)
    if len(queue) > 10:
        return await msg.reply_text(
            get_string("queue_limit", lang).format(count=len(queue))
        )

    await load_admin_cache(c, chat_id)
    if not await is_admin(chat_id, c.me.id):
        return await msg.reply_text(get_string("need_admin_reload", lang))

    reply = await msg.getRepliedMessage() if msg.reply_to_message_id else None
    url = await get_url(msg, reply)
    args = extract_argument(msg.text)

    reply_message = await msg.reply_text(get_string("searching", lang))
    if isinstance(reply_message, types.Error):
        LOGGER.warning("Error sending reply: %s", reply_message)
        return None

    # Assistant checks
    ub = await call.get_client(chat_id)
    if isinstance(ub, (types.Error, NoneType)):
        return await edit_text(reply_message, text=ub.message)

    # User status check
    user_key = f"{chat_id}:{ub.me.id}"
    user_status = user_status_cache.get(user_key) or await check_user_status(
        c, chat_id, ub.me.id
    )

    if isinstance(user_status, types.Error):
        return await edit_text(reply_message, f"❌ {str(user_status)}")

    if user_status.getType() in {
        types.ChatMemberStatusLeft().getType(),
        types.ChatMemberStatusBanned().getType(),
        types.ChatMemberStatusRestricted().getType(),
    }:
        if user_status == types.ChatMemberStatusBanned().getType():
            await unban_ub(c, chat_id, ub.me.id)
        join = await join_ub(chat_id, c, ub)
        if isinstance(join, types.Error):
            return await edit_text(reply_message, f"❌ {join.message}")

    await del_msg(msg)
    wrapper = (YouTubeData if is_video else MusicServiceWrapper)(url or args)

    # No args or reply
    if not args and not url and not (reply and Telegram(reply).is_valid()):
        if is_video:
            return await edit_text(
                reply_message,
                text=get_string("usage_video", lang),
                reply_markup=SupportButton,
            )
        else:
            return await _handle_recommendations(c, reply_message, wrapper)

    user_by = await msg.mention()

    # Telegram file support
    if reply and Telegram(reply).is_valid():
        return await _handle_telegram_file(c, channel, reply, reply_message, user_by)

    if url:
        if not wrapper.is_valid(url):
            return await edit_text(
                reply_message,
                get_string("invalid_url", lang),
                reply_markup=SupportButton,
            )

        if song := await wrapper.get_info():
            return await play_music(c, reply_message, song, user_by, channel, is_video=is_video)

        return await edit_text(
            reply_message,
            get_string("failed_song_info", lang),
            reply_markup=SupportButton,
        )

    # Search
    if is_video:
        search = await wrapper.search()
        if not search or not search.tracks:
            return await edit_text(
                reply_message,
                get_string("no_results", lang),
                reply_markup=SupportButton,
            )

        if song := await MusicServiceWrapper(search.tracks[0].url).get_info():
            return await play_music(c, reply_message, song, user_by, channel, is_video=True)

        return await edit_text(
            reply_message,
            get_string("failed_song_info", lang),
            reply_markup=SupportButton,
        )
    else:
        return await _handle_text_search(c, reply_message, channel, wrapper, user_by)


@Client.on_message(filters=Filter.command(["play", "cplay"]))
async def play_audio(c: Client, msg: types.Message) -> None:
    await handle_play_command(c, msg, False)


@Client.on_message(filters=Filter.command(["vplay", "cvplay"]))
async def play_video(c: Client, msg: types.Message) -> None:
    await handle_play_command(c, msg, True)

@Client.on_message(filters=Filter.command(["direct", "cdirect"]))
async def play_file(_: Client, msg: types.Message) -> None:
    """Play a direct link. JUST FOR TESTING"""
    is_channel = is_channel_cmd(msg.text)
    chat_id = await db.get_channel_id(msg.chat_id) if is_channel else msg.chat_id
    channel = ChannelPlay(
        chat_id=chat_id,
        is_channel=is_channel and chat_id != msg.chat_id,
    )

    lang = await db.get_lang(msg.chat_id)
    if chat_id > 0:
        await msg.reply_text(get_string("only_supergroup", lang))
        return None

    if chat_cache.is_active(chat_id):
        await msg.reply_text(f"first stop (/end) the song: {chat_cache.get_queue(chat_id)[0].name}")
        return None

    if not await is_owner(msg.chat_id, msg.from_id):
        await msg.reply_text(get_string("only_owner", lang))
        return None

    link = extract_argument(msg.text)
    if not link:
        await msg.reply_text("Give me an direct playable link to play.")
        return None

    _call = await call.play_media(chat_id, link, True)
    if isinstance(_call, types.Error):
        await edit_text(msg, text=f"⚠️ {str(_call)}")
        return None

    chat_cache.add_song(chat_id, CachedTrack(name="", artist="", track_id="", loop=0, duration=0, file_path=link, thumbnail="", user="", platform="", is_video=True, url=link, channel=channel))
    return None
