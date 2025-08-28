#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re

from pytdbot import Client, types

from TgMusic.core import YouTubeData, DownloaderWrapper, db, call, tg
from TgMusic.core import (
    CachedTrack,
    MusicTrack,
    PlatformTracks,
    chat_cache,
)
from TgMusic.logger import LOGGER
from TgMusic.core import (
    Filter,
    SupportButton,
    control_buttons,
)
from TgMusic.core import admins_only
from TgMusic.modules.utils import sec_to_min, get_audio_duration
from TgMusic.modules.utils.play_helpers import (
    del_msg,
    edit_text,
    extract_argument,
    get_url,
)
from TgMusic.core.thumbnails import gen_thumb


def _get_jiosaavn_url(track_id: str) -> str:
    """Generate JioSaavn URL from track ID."""
    try:
        title, song_id = track_id.rsplit("/", 1)
    except ValueError:
        return ""
    title = re.sub(r'[\(\)"\',]', "", title.lower()).replace(" ", "-")
    return f"https://www.jiosaavn.com/song/{title}/{song_id}"


def _get_platform_url(platform: str, track_id: str) -> str:
    """Generate platform URL from track ID based on platform."""
    platform = platform.lower()
    if not track_id:
        return ""

    platform_urls = {
        "youtube": f"https://youtube.com/watch?v={track_id}",
        "spotify": f"https://open.spotify.com/track/{track_id}",
        "jiosaavn": _get_jiosaavn_url(track_id),
    }
    return platform_urls.get(platform, "")


def build_song_selection_message(
    user_by: str, tracks: list[MusicTrack]
) -> tuple[str, types.ReplyMarkupInlineKeyboard]:
    """Build interactive song selection message with inline keyboard."""
    greeting = f"{user_by}, select a track:" if user_by else "Select a track:"
    buttons = [
        [
            types.InlineKeyboardButton(
                text=f"{track.name[:18]} - {track.artist}",
                type=types.InlineKeyboardButtonTypeCallback(
                    f"vcplay_{track.platform.lower()}_{track.id}".encode()
                ),
            )
        ]
        for track in tracks[:4]  # Show first 4 results
    ]
    return greeting, types.ReplyMarkupInlineKeyboard(buttons)


async def _update_msg_with_thumb(
    c: Client,
    msg: types.Message,
    text: str,
    thumb: str,
    button: types.ReplyMarkupInlineKeyboard,
):
    """Update message with thumbnail if available."""
    if not thumb:
        return await edit_text(
            msg, text=text, reply_markup=button, disable_web_page_preview=True
        )

    parsed_text = await c.parseTextEntities(text, types.TextParseModeHTML())
    if isinstance(parsed_text, types.Error):
        return await edit_text(msg, text=parsed_text.message, reply_markup=button)

    input_content = types.InputMessagePhoto(
        types.InputFileLocal(thumb), caption=parsed_text
    )
    edit_result = await c.editMessageMedia(
        chat_id=msg.chat_id,
        message_id=msg.id,
        input_message_content=input_content,
        reply_markup=button,
    )

    return edit_result


async def _handle_single_track(
    c: Client,
    msg: types.Message,
    track: MusicTrack,
    user_by: str,
    file_path: str = None,
    is_video: bool = False,
):
    chat_id = msg.chat_id
    song = CachedTrack(
        name=track.name,
        track_id=track.id,
        loop=0,
        duration=track.duration,
        file_path=file_path or "",
        thumbnail=track.cover,
        user=user_by,
        platform=track.platform,
        is_video=is_video,
        url=track.url,
    )

    # Download track if not already cached
    if not song.file_path:
        download_result = await call.song_download(song)
        if isinstance(download_result, types.Error):
            return await edit_text(
                msg, f"❌ Download failed: {download_result.message}"
            )

        song.file_path = download_result
        if not download_result:
            return await edit_text(msg, "❌ Failed to download track")

    # Get duration if not provided
    song.duration = song.duration or await get_audio_duration(song.file_path)

    if chat_cache.is_active(chat_id):
        # Add to queue if playback is active
        queue = chat_cache.get_queue(chat_id)
        chat_cache.add_song(chat_id, song)

        queue_info = (
            f"<b>🎧 Added to Queue (#{len(queue)})</b>\n\n"
            f"▫ <b>Track:</b> <a href='{song.url}'>{song.name}</a>\n"
            f"▫ <b>Duration:</b> {sec_to_min(song.duration)}\n"
            f"▫ <b>Requested by:</b> {song.user}"
        )

        thumb = await gen_thumb(song) if await db.get_thumbnail_status(chat_id) else ""
        return await _update_msg_with_thumb(
            c,
            msg,
            queue_info,
            thumb,
            control_buttons("play") if await db.get_buttons_status(chat_id) else None,
        )

    # Start new playback session
    chat_cache.set_active(chat_id, True)
    chat_cache.add_song(chat_id, song)

    play_result = await call.play_media(chat_id, song.file_path, video=is_video)
    if isinstance(play_result, types.Error):
        return await edit_text(msg, text=f"⚠️ Playback error: {play_result.message}")

    # Prepare now playing message
    thumb = await gen_thumb(song) if await db.get_thumbnail_status(chat_id) else ""
    now_playing = (
        f"🎵 <b>Now Playing:</b>\n\n"
        f"▫ <b>Track:</b> <a href='{song.url}'>{song.name}</a>\n"
        f"▫ <b>Duration:</b> {sec_to_min(song.duration)}\n"
        f"▫ <b>Requested by:</b> {song.user}"
    )

    update_result = await _update_msg_with_thumb(
        c,
        msg,
        now_playing,
        thumb,
        control_buttons("play") if await db.get_buttons_status(chat_id) else None,
    )

    if isinstance(update_result, types.Error):
        LOGGER.warning("Message update failed: %s", update_result)
    return None


async def _handle_multiple_tracks(
    msg: types.Message, tracks: list[MusicTrack], user_by: str
):
    """Process and queue multiple tracks (playlist/album)."""
    chat_id = msg.chat_id
    is_active = chat_cache.is_active(chat_id)
    queue = chat_cache.get_queue(chat_id)

    queue_header = "<b>📥 Added to Queue:</b>\n<blockquote expandable>\n"
    queue_items = []

    for index, track in enumerate(tracks):
        position = len(queue) + index
        chat_cache.add_song(
            chat_id,
            CachedTrack(
                name=track.name,
                track_id=track.id,
                loop=1 if not is_active and index == 0 else 0,
                duration=track.duration,
                thumbnail=track.cover,
                user=user_by,
                file_path="",
                platform=track.platform,
                is_video=False,
                url=track.url,
            ),
        )
        queue_items.append(
            f"<b>{position}.</b> {track.name}\n└ Duration: {sec_to_min(track.duration)}"
        )

    queue_summary = (
        f"</blockquote>\n"
        f"<b>📋 Total in Queue:</b> {len(chat_cache.get_queue(chat_id))}\n"
        f"<b>⏱ Total Duration:</b> {sec_to_min(sum(t.duration for t in tracks))}\n"
        f"<b>👤 Requested by:</b> {user_by}"
    )

    full_message = queue_header + "\n".join(queue_items) + queue_summary

    # Handle message length limit
    if len(full_message) > 4096:
        full_message = queue_summary

    if not is_active:
        await call.play_next(chat_id)

    await edit_text(msg, full_message, reply_markup=control_buttons("play"))


async def play_music(
    c: Client,
    msg: types.Message,
    url_data: PlatformTracks,
    user_by: str,
    tg_file_path: str = None,
    is_video: bool = False,
):
    """Main music playback handler for both single tracks and playlists."""
    if not url_data or not url_data.tracks:
        return await edit_text(msg, "❌ No tracks found in the provided source.")

    await edit_text(msg, text="⬇️ Downloading track...")

    if len(url_data.tracks) == 1:
        return await _handle_single_track(
            c, msg, url_data.tracks[0], user_by, tg_file_path, is_video
        )
    return await _handle_multiple_tracks(msg, url_data.tracks, user_by)


async def _handle_telegram_file(
    c: Client, reply: types.Message, reply_message: types.Message, user_by: str
):
    """Process Telegram audio/video file attachments."""
    content = reply.content
    is_video = isinstance(content, (types.MessageVideo, types.Video)) or (
        isinstance(content, (types.MessageDocument, types.Document))
        and getattr(content, "mime_type", "").startswith("video/")
    )

    # Download the attached file
    file_path, file_name = await tg.download_msg(reply, reply_message)
    if isinstance(file_path, types.Error):
        return await edit_text(
            reply_message,
            text=(
                "<b>⚠️ Download Failed</b>\n\n"
                f"▫ <b>File:</b> <code>{file_name}</code>\n"
                f"▫ <b>Error:</b> <code>{file_path.message}</code>"
            ),
        )

    duration = await get_audio_duration(file_path.path)
    track_data = PlatformTracks(
        tracks=[
            MusicTrack(
                name=file_name,
                id=reply.remote_unique_file_id,
                cover="",
                duration=duration,
                url="",
                platform="telegram",
            )
        ]
    )

    await play_music(c, reply_message, track_data, user_by, file_path.path, is_video)
    return None


async def _handle_text_search(
    c: Client,
    msg: types.Message,
    wrapper: DownloaderWrapper,
    user_by: str,
):
    """Handle text-based music searches."""
    chat_id = msg.chat_id
    play_type = await db.get_play_type(chat_id)

    search_result = await wrapper.search()
    if isinstance(search_result, types.Error):
        return await edit_text(
            msg,
            text=f"🔍 Search failed: {search_result.message}",
            reply_markup=SupportButton,
        )

    if not search_result or not search_result.tracks:
        return await edit_text(
            msg,
            text="🔍 No results found. Try different keywords.",
            reply_markup=SupportButton,
        )

    # Direct play if configured
    if play_type == 0:
        track_url = search_result.tracks[0].url
        track_info = await DownloaderWrapper(track_url).get_info()
        if isinstance(track_info, types.Error):
            return await edit_text(
                msg,
                text=f"⚠️ Track info error: {track_info.message}",
                reply_markup=SupportButton,
            )
        return await play_music(c, msg, track_info, user_by)

    # Show selection menu
    selection_text, selection_keyboard = build_song_selection_message(
        user_by, search_result.tracks
    )
    await edit_text(
        msg,
        text=selection_text,
        reply_markup=selection_keyboard,
        disable_web_page_preview=True,
    )
    return None


async def handle_play_command(c: Client, msg: types.Message, is_video: bool = False):
    """Main handler for /play and /vplay commands."""
    chat_id = msg.chat_id
    # Validate chat type
    if chat_id > 0:
        return await msg.reply_text("❌ This command only works in groups/channels.")

    # Check queue limit
    queue = chat_cache.get_queue(chat_id)
    if len(queue) > 10:
        return await msg.reply_text(
            "⚠️ Queue limit reached (10 tracks max). Use /end to clear queue."
        )

    reply = await msg.getRepliedMessage() if msg.reply_to_message_id else None
    url = await get_url(msg, reply)

    tg_pubic_url = url and re.fullmatch(r"https:\/\/t\.me\/([a-zA-Z0-9_]{5,})\/(\d+)", url)
    if not reply and tg_pubic_url:
        info = await c.getMessageLinkInfo(url)
        if isinstance(info, types.Error) or not info.message:
            await msg.reply_text(f"⚠️ Could not resolve message from link. {info.message}")
            c.logger.warning(f"❌ Could not resolve message from link: {url}; {info}")
            return None
        reply = await c.getMessage(info.chat_id, info.message.id)

    # Send initial response
    status_msg = await msg.reply_text("🔍 Processing request...")
    if isinstance(status_msg, types.Error):
        LOGGER.error("Failed to send status message: %s", status_msg)
        return None

    c.loop.create_task(del_msg(msg)) # Clean up command message

    args = extract_argument(msg.text)
    # Initialize appropriate downloader
    wrapper = (YouTubeData if is_video else DownloaderWrapper)(url or args)

    # Validate input
    if not args and not url and (not reply or not tg.is_valid(reply)):
        usage_text = (
            "🎵 <b>Usage:</b>\n"
            f"/{'vplay' if is_video else 'play'} [song_name|URL]\n\n"
            "Supported platforms:\n"
            "▫ YouTube\n▫ Spotify\n▫ JioSaavn\n▫ SoundCloud\n▫ Apple Music"
        )
        return await edit_text(status_msg, text=usage_text, reply_markup=SupportButton)

    requester = await msg.mention()

    # Handle Telegram file attachments
    if reply and tg.is_valid(reply):
        return await _handle_telegram_file(c, reply, status_msg, requester)

    # Handle URL playback
    if url:
        if not wrapper.is_valid():
            return await edit_text(
                status_msg,
                text=(
                    "⚠️ Unsupported URL\n\n"
                    "Supported platforms:\n"
                    "▫ YouTube\n▫ Spotify\n▫ JioSaavn\n▫ SoundCloud\n▫ Apple Music"
                ),
                reply_markup=SupportButton,
            )

        track_info = await wrapper.get_info()
        if isinstance(track_info, types.Error):
            return await edit_text(
                status_msg,
                text=f"⚠️ Couldn't retrieve track info:\n{track_info.message}",
                reply_markup=SupportButton,
            )

        return await play_music(c, status_msg, track_info, requester, is_video=is_video)

    # Handle text search for audio only
    if not is_video:
        return await _handle_text_search(c, status_msg, wrapper, requester)

    # Handle video search
    search_result = await wrapper.search()
    if isinstance(search_result, types.Error):
        return await edit_text(
            status_msg,
            text=f"🔍 Search failed: {search_result.message}",
            reply_markup=SupportButton,
        )

    if not search_result or not search_result.tracks:
        return await edit_text(
            status_msg,
            text="🔍 No results found. Try different keywords.",
            reply_markup=SupportButton,
        )

    # Play first video result
    video_info = await DownloaderWrapper(search_result.tracks[0].url).get_info()
    if isinstance(video_info, types.Error):
        return await edit_text(
            status_msg,
            text=f"⚠️ Video error: {video_info.message}",
            reply_markup=SupportButton,
        )

    return await play_music(c, status_msg, video_info, requester, is_video=True)


@Client.on_message(filters=Filter.command("play"), position=-5)
@admins_only(permissions="can_invite_users", is_bot=True)
async def play_audio(c: Client, msg: types.Message) -> None:
    """Audio playback command handler."""
    await handle_play_command(c, msg, False)


@Client.on_message(filters=Filter.command("vplay"), position=-4)
@admins_only(permissions="can_invite_users", is_bot=True)
async def play_video(c: Client, msg: types.Message) -> None:
    """Video playback command handler."""
    await handle_play_command(c, msg, True)
