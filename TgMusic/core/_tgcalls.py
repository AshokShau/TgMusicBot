#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import os
import random
import re
from pathlib import Path
from typing import Optional, Union

import ntgcalls
from ntgcalls import ConnectionNotFound
from pyrogram import Client as PyroClient
from pyrogram import errors
from pytdbot import Client, types
from pytgcalls import PyTgCalls, exceptions
from pytgcalls.types import (
    AudioQuality,
    CallConfig,
    ChatUpdate,
    GroupCallConfig,
    MediaStream,
    Update,
    UpdatedGroupCallParticipant,
    VideoQuality,
    stream,
)

from TgMusic.logger import LOGGER
from TgMusic.modules.utils import (
    get_audio_duration,
    sec_to_min,
)

from ._cacher import (
    ChatMemberStatusResult,
    chat_cache,
    chat_invite_cache,
    user_status_cache,
)
from ._config import config
from ._database import db
from ._dataclass import CachedTrack
from ._downloader import DownloaderWrapper
from .buttons import control_buttons
from .thumbnails import gen_thumb
from .utils import send_logger


class Calls:
    def __init__(self):
        self.calls: dict[str, PyTgCalls] = {}
        self.pyrogram_clients: dict[str, PyroClient] = {}
        self.client_counter: int = 1
        self.available_clients: list[str] = []
        self.bot: Optional[Client] = None

    async def add_bot(self, bot: Client) -> types.Ok:
        self.bot = bot
        return types.Ok()

    async def _get_client_name(self, chat_id: int) -> Union[str, types.Error]:
        """Get an available client session for a chat."""
        if not self.available_clients:
            return types.Error(
                code=500, message="No clients available\nReport this issue"
            )

        if chat_id == 1:
            return random.choice(self.available_clients)

        assistant = await db.get_assistant(chat_id)
        if assistant and assistant in self.available_clients:
            return assistant

        new_client = random.choice(self.available_clients)
        await db.set_assistant(chat_id, assistant=new_client)
        LOGGER.info("Set assistant for %s to %s", chat_id, new_client)
        return new_client

    async def _group_assistant(self, chat_id: int) -> Union[PyTgCalls, types.Error]:
        client_name = await self._get_client_name(chat_id)
        if isinstance(client_name, types.Error):
            return client_name

        return self.calls[client_name]

    async def get_client(self, chat_id: int) -> Union[PyroClient, types.Error]:
        """Get the pyrogram client instance for a chat.

        Args:
            chat_id: Target chat ID

        Returns:
            PyroClient instance or types.Error if unavailable
        """
        client = await self._group_assistant(chat_id)
        if isinstance(client, types.Error):
            return client

        ub: PyroClient = client.mtproto_client
        if ub is None or not hasattr(ub, "me") or ub.me is None:
            return types.Error(
                code=500,
                message="Client session not initialized properly. "
                "Please report this issue.",
            )

        if ub.me.is_bot:
            return types.Error(
                code=500,
                message="Client session is a bot account, Please use user account.",
            )
        return ub

    async def start_client(
        self, api_id: int, api_hash: str, session_string: str
    ) -> None:
        """Start a new pyrogram client session.

        Args:
            api_id: Telegram API ID
            api_hash: Telegram API hash
            session_string: Session string for authentication
        """
        client_name = f"client{self.client_counter}"
        try:
            user_bot = PyroClient(
                client_name,
                api_id=api_id,
                api_hash=api_hash,
                session_string=session_string,
                fetch_replies=False,
                fetch_stories=False,
                fetch_topics=False,
                fetch_stickers=False,
                no_updates=config.NO_UPDATES,
            )
            calls = PyTgCalls(user_bot, cache_duration=100)
            self.calls[client_name] = calls
            self.available_clients.append(client_name)
            self.pyrogram_clients[client_name] = user_bot
            self.client_counter += 1
            await calls.start()
            LOGGER.info("Client %s started successfully", client_name)
        except Exception as e:
            LOGGER.error("Error starting client %s: %s", client_name, e)
            raise RuntimeError(f"Failed to start client {client_name}: {str(e)}") from e

    async def stop_all_clients(self) -> None:
        for name, client in self.pyrogram_clients.items():
            try:
                if client.is_connected:
                    LOGGER.info("Stopping client %s", name)
                    await client.stop()
            except Exception as e:
                LOGGER.error("Error stopping client %s: %s", name, e)

    async def register_decorators(self) -> None:
        """Register pytgcalls event handlers."""
        for _call in self.calls.values():

            @_call.on_update()
            async def general_handler(_, update: Update, _call=_call):
                try:
                    if isinstance(update, stream.StreamEnded):
                        await self.play_next(update.chat_id)
                    elif isinstance(update, UpdatedGroupCallParticipant):
                        return
                    elif isinstance(update, ChatUpdate) and (
                        update.status.KICKED or update.status.LEFT_GROUP
                    ):
                        LOGGER.debug(
                            "Cleaning up chat %s after leaving", update.chat_id
                        )
                        chat_cache.clear_chat(update.chat_id)
                    elif (
                        isinstance(update, ChatUpdate)
                        and update.status.CLOSED_VOICE_CHAT
                    ):
                        LOGGER.debug(
                            "Cleaning up chat %s after leaving", update.chat_id
                        )
                        chat_cache.clear_chat(update.chat_id)
                        await self.end(update.chat_id)
                except Exception as e:
                    LOGGER.error("Error in general handler: %s", e, exc_info=True)

    async def play_media(
        self,
        chat_id: int,
        file_path: Union[str, Path],
        video: bool = False,
        ffmpeg_parameters: Optional[str] = None,
    ) -> Union[types.Ok, types.Error]:
        """Play media in a voice chat.

        Args:
            chat_id: Target chat ID
            file_path: Path to media file
            video: Whether to stream video
            ffmpeg_parameters: Custom ffmpeg parameters

        Returns:
            types.Ok on success or types.Error on failure
        """
        LOGGER.info(
            "Playing media for chat %s: %s (video=%s)", chat_id, file_path, video
        )

        client = await self._group_assistant(chat_id)
        if isinstance(client, types.Error):
            chat_cache.clear_chat(chat_id)
            return client

        # Validate media file exists if not URL
        if not re.match("^https?://", str(file_path)) and not os.path.exists(file_path):
            return types.Error(
                code=404, message="Media file not found. It may have been deleted."
            )

        join = await self._join_assistant(chat_id)
        if isinstance(join, types.Error):
            chat_cache.clear_chat(chat_id)
            return join

        _stream = MediaStream(
            audio_path=file_path,
            media_path=file_path,
            audio_parameters=AudioQuality.HIGH if video else AudioQuality.STUDIO,
            video_parameters=VideoQuality.FHD_1080p if video else VideoQuality.SD_360p,
            audio_flags=MediaStream.Flags.REQUIRED,
            video_flags=(
                MediaStream.Flags.AUTO_DETECT if video else MediaStream.Flags.IGNORE
            ),
            ffmpeg_parameters=ffmpeg_parameters,
        )

        call_config = (
            GroupCallConfig(auto_start=False) if chat_id < 0 else CallConfig(timeout=50)
        )
        try:
            await client.play(chat_id, _stream, call_config)
            # Send playback log if enabled
            if await db.get_logger_status(self.bot.me.id):
                self.bot.loop.create_task(
                    send_logger(
                        self.bot, chat_id, chat_cache.get_playing_track(chat_id)
                    )
                )

            return types.Ok()
        except (exceptions.NoActiveGroupCall, ntgcalls.ConnectionNotFound):
            return types.Error(
                code=404,
                message="No active voice chat found.\n\n"
                "Please start a voice chat and try again.",
            )
        except ntgcalls.ConnectionError as e:
            LOGGER.error("Connection error during playback: %s", e)
            return types.Error(
                code=502,
                message="Connection error detected. Please try again later.\nDid you just close the voice chat?",
            )
        except ntgcalls.TelegramServerError:
            LOGGER.warning("Telegram server error during playback")
            return types.Error(
                code=502,
                message="Telegram server issues detected. Please try again later.",
            )
        except exceptions.NoAudioSourceFound as e:
            LOGGER.error("Audio source not found in chat %s: %s", chat_id, str(e))
            return types.Error(code=404, message="Audio source not found.")
        except FileNotFoundError:
            return types.Error(code=404, message="File not found.")
        except errors.RPCError as e:
            LOGGER.error("Playback failed in chat %s: %s", chat_id, str(e))
            return types.Error(code=e.CODE or 500, message=f"Playback error: {str(e)}")
        except Exception as e:
            LOGGER.error(
                "Playback failed in chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Playback error: {str(e)}")

    async def play_next(self, chat_id: int) -> None:
        """Handle playback of next track in queue.

        Args:
            chat_id: Target chat ID

        Handles:
            - Loop counts
            - Queue management
            - Empty queue scenarios
        """
        LOGGER.info("Playing next song for chat %s", chat_id)
        loop = chat_cache.get_loop_count(chat_id)
        if loop > 0:
            chat_cache.set_loop_count(chat_id, loop - 1)
            if current_song := chat_cache.get_playing_track(chat_id):
                await self._play_song(chat_id, current_song)
                return

        # Get next song from queue
        if next_song := chat_cache.get_upcoming_track(chat_id):
            chat_cache.remove_current_song(chat_id)
            await self._play_song(chat_id, next_song)
        else:
            chat_cache.remove_current_song(chat_id)
            await self._handle_no_songs(chat_id)

    async def _play_song(self, chat_id: int, song: CachedTrack) -> None:
        """Internal method to play a specific song.

        Args:
            chat_id: Target chat ID
            song: CachedTrack object containing song data
        """
        LOGGER.info("Playing song for chat %s: %s", chat_id, song.name)

        try:
            # Send an initial loading message
            reply = await self.bot.sendTextMessage(
                chat_id, "⏳ Loading... Please wait."
            )
            if isinstance(reply, types.Error):
                LOGGER.error("Failed to send message: %s", reply)
                return

            # Download song if isn't downloaded
            file_path = song.file_path or await self.song_download(song)
            if not file_path:
                await reply.edit_text(
                    "⚠️ Failed to download the song.\n" "Skipping to next track..."
                )
                await self.play_next(chat_id)
                return

            # Start playback
            play_result = await self.play_media(chat_id, file_path, video=song.is_video)
            if isinstance(play_result, types.Error):
                await reply.edit_text(play_result.message)
                return

            # Get duration if not available
            duration = song.duration or await get_audio_duration(file_path)

            # Prepare a playback message
            text = (
                f"<b>Now Playing:</b>\n\n"
                f"‣ <b>Title:</b> <a href='{song.url}'>{song.name}</a>\n"
                f"‣ <b>Duration:</b> {sec_to_min(duration)}\n"
                f"‣ <b>Requested by:</b> {song.user}"
            )

            thumbnail = (
                await gen_thumb(song) if await db.get_thumbnail_status(chat_id) else ""
            )
            # Parse text entities
            parse = await self.bot.parseTextEntities(text, types.TextParseModeHTML())
            if isinstance(parse, types.Error):
                LOGGER.error("Failed to parse text entities: %s", parse)
                parse = text  # Fallback to an original text

            # Update a message with media or text
            if thumbnail:
                input_content = types.InputMessagePhoto(
                    photo=types.InputFileLocal(thumbnail), caption=parse
                )
                await self.bot.editMessageMedia(
                    chat_id=chat_id,
                    message_id=reply.id,
                    input_message_content=input_content,
                    reply_markup=(
                        control_buttons("play")
                        if await db.get_buttons_status(chat_id)
                        else None
                    ),
                )
            else:
                await self.bot.editMessageText(
                    chat_id=chat_id,
                    message_id=reply.id,
                    input_message_content=types.InputMessageText(
                        text=parse,
                        link_preview_options=types.LinkPreviewOptions(is_disabled=True),
                    ),
                    reply_markup=(
                        control_buttons("play")
                        if await db.get_buttons_status(chat_id)
                        else None
                    ),
                )

        except Exception as e:
            LOGGER.error(
                "Error in _play_song for chat %s: %s", chat_id, str(e), exc_info=True
            )

    @staticmethod
    async def song_download(song: CachedTrack) -> Union[Path, types.Error]:
        """Download a song from various platforms.

        Args:
            song: CachedTrack object containing song data

        Returns:
            Path to the downloaded file or types.Error if download fails
        """
        song_url = song.url
        wrapper = DownloaderWrapper(song_url)
        if wrapper.is_valid():
            track_info = await wrapper.get_track()
            if isinstance(track_info, types.Error):
                return track_info

            return await wrapper.download_track(track_info, song.is_video)
        return types.Error(
            code=400,
            message=f"Invalid URL: {song_url}",
        )

    async def _handle_no_songs(self, chat_id: int) -> None:
        """Handle an empty queue scenario.

        Args:
            chat_id: Target chat ID
        """
        await self.end(chat_id)
        await self.bot.sendTextMessage(
            chat_id, text="🎵 Queue finished.\nUse /play to add more songs!"
        )

    async def end(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """End playback and clean up for a chat.

        Args:
            chat_id: Target chat ID

        Returns:
            types.Ok on success or types.Error on failure
        """
        LOGGER.info("Ending playback for chat %s", chat_id)
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            chat_cache.clear_chat(chat_id)
            try:
                await client.leave_call(chat_id)
            except (
                exceptions.NotInCallError,
                errors.GroupCallInvalid,
                exceptions.NoActiveGroupCall,
                ConnectionNotFound,
                errors.GroupcallForbidden,
            ):
                pass  # Already not in call

            return types.Ok()
        except Exception as e:
            LOGGER.error(
                "Error ending call for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Failed to end call: {str(e)}")

    async def seek_stream(
        self,
        chat_id: int,
        file_path_or_url: Union[str, Path],
        to_seek: int,
        duration: int,
        is_video: bool,
    ) -> Union[types.Ok, types.Error]:
        """Seek to a position in the current stream.

        Args:
            chat_id: Target chat ID
            file_path_or_url: Media file path or URL
            to_seek: Position to seek to (seconds)
            duration: Total duration to play (seconds)
            is_video: Whether the stream is video

        Returns:
            types.Ok on success or types.Error on failure
        """
        if to_seek < 0 or duration <= 0:
            return types.Error(
                code=400,
                message="Invalid seek position or duration.\n"
                "Position must be positive and duration must be greater than 0.",
            )

        try:
            is_url = bool(re.match(r"https?://", str(file_path_or_url)))
            ffmpeg_params = (
                f"-ss {to_seek} -i {file_path_or_url} -to {duration}"
                if is_url or not os.path.isfile(file_path_or_url)
                else f"-ss {to_seek} -to {duration}"
            )

            return await self.play_media(
                chat_id, file_path_or_url, is_video, ffmpeg_params
            )
        except Exception as e:
            LOGGER.error("Seek failed for chat %s: %s", chat_id, str(e), exc_info=True)
            return types.Error(code=500, message=f"Seek operation failed: {str(e)}")

    async def speed_change(
        self, chat_id: int, speed: float = 1.0
    ) -> Union[types.Ok, types.Error]:
        """Change playback speed.

        Args:
            chat_id: Target chat ID
            speed: Playback speed (0.5-4.0)

        Returns:
            types.Ok on success or types.Error on failure
        """
        if not 0.5 <= speed <= 4.0:
            return types.Error(
                code=400, message="Invalid speed value.\n" "Must be between 0.5 and 4.0"
            )

        curr_song = chat_cache.get_playing_track(chat_id)
        if not curr_song or not curr_song.file_path:
            return types.Error(code=404, message="No track currently playing")

        return await self.play_media(
            chat_id,
            curr_song.file_path,
            curr_song.is_video,
            ffmpeg_parameters=(
                f"-atend -filter:v setpts=0.5*PTS " f"-filter:a atempo={speed}"
            ),
        )

    async def change_volume(
        self, chat_id: int, volume: int
    ) -> Union[None, types.Error]:
        """Change playback volume.

        Args:
            chat_id: Target chat ID
            volume: Volume level (1-200)

        Returns:
            None on success or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            if volume < 1 or volume > 200:
                return types.Error(code=400, message="Volume must be between 1 and 200")

            await client.change_volume_call(chat_id, volume)
            return None
        except Exception as e:
            LOGGER.error(
                "Volume change failed for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Volume change failed: {str(e)}")

    async def mute(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """Mute the current stream.

        Args:
            chat_id: Target chat ID

        Returns:
            types.Ok on success or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            await client.mute(chat_id)
            return types.Ok()
        except (exceptions.NotInCallError, ConnectionNotFound):
            return types.Error(code=400, message="My Assistant is not in a call")
        except Exception as e:
            LOGGER.error("Mute failed for chat %s: %s", chat_id, str(e), exc_info=True)
            return types.Error(code=500, message=f"Mute operation failed: {str(e)}")

    async def unmute(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """Unmute the current stream.

        Args:
            chat_id: Target chat ID

        Returns:
            types.Ok on success or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            await client.unmute(chat_id)
            return types.Ok()
        except (exceptions.NotInCallError, ConnectionNotFound):
            return types.Error(code=400, message="My Assistant is not in a call")
        except Exception as e:
            LOGGER.error(
                "Unmute failed for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Unmute operation failed: {str(e)}")

    async def resume(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """Resume a paused stream.

        Args:
            chat_id: Target chat ID

        Returns:
            types.Ok on success or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            await client.resume(chat_id)
            return types.Ok()
        except (exceptions.NotInCallError, ConnectionNotFound):
            return types.Error(code=400, message="My Assistant is not in a call")
        except Exception as e:
            LOGGER.error(
                "Resume failed for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Resume operation failed: {str(e)}")

    async def pause(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """Pause the current stream.

        Args:
            chat_id: Target chat ID

        Returns:
            types.Ok on success or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            await client.pause(chat_id)
            return types.Ok()
        except Exception as e:
            LOGGER.error("Pause failed for chat %s: %s", chat_id, str(e), exc_info=True)
            return types.Error(code=500, message=f"Pause operation failed: {str(e)}")

    async def played_time(self, chat_id: int) -> Union[int, types.Error]:
        """Get the current playback position.

        Args:
            chat_id: Target chat ID

        Returns:
            Current position in seconds or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            return await client.time(chat_id)
        except exceptions.NotInCallError:
            chat_cache.clear_chat(chat_id)
            return 0
        except Exception as e:
            LOGGER.error(
                "Time check failed for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(
                code=500, message=f"Failed to get playback time: {str(e)}"
            )

    async def vc_users(self, chat_id: int) -> Union[list, types.Error]:
        """Get a list of participants in voice chat.

        Args:
            chat_id: Target chat ID

        Returns:
            List of participants or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            return await client.get_participants(chat_id)
        except exceptions.UnsupportedMethod:
            return types.Error(
                code=501, message="This method is not supported by the server"
            )
        except Exception as e:
            LOGGER.error(
                "Participant list failed for chat %s: %s",
                chat_id,
                str(e),
                exc_info=True,
            )
            return types.Error(
                code=500, message=f"Failed to get participants: {str(e)}"
            )

    async def stats_call(self, chat_id: int) -> Union[tuple[float, float], types.Error]:
        """Get call statistics.

        Args:
            chat_id: Target chat ID

        Returns:
            Tuple of (ping, cpu_usage) or types.Error on failure
        """
        try:
            client = await self._group_assistant(chat_id)
            if isinstance(client, types.Error):
                return client

            return (
                client.ping,
                await client.cpu_usage,
            )
        except Exception as e:
            LOGGER.error(
                "Stats check failed for chat %s: %s", chat_id, str(e), exc_info=True
            )
            return types.Error(code=500, message=f"Failed to get stats: {str(e)}")

    async def check_user_status(
        self, chat_id: int
    ) -> Union[ChatMemberStatusResult, types.Error]:
        client = await self.get_client(chat_id)
        if isinstance(client, types.Error):
            LOGGER.error(f"Failed to get client for chat {chat_id}")
            return client

        user_id = client.me.id
        cache_key = f"{chat_id}:{user_id}"
        user_status = user_status_cache.get(cache_key)
        if not user_status:
            user = await self.bot.getChatMember(
                chat_id=chat_id, member_id=types.MessageSenderUser(user_id)
            )
            if isinstance(user, types.Error):
                return types.ChatMemberStatusLeft() if user.code == 400 else user

            if user.status is None:
                return types.ChatMemberStatusLeft()

            user_status = user.status
            user_status_cache[cache_key] = user_status

        return user_status

    async def _join_assistant(self, chat_id: int) -> Union[types.Ok, types.Error]:
        user_status = await self.check_user_status(chat_id)
        if isinstance(user_status, types.Error):
            return user_status

        if user_status.getType() in {
            types.ChatMemberStatusLeft().getType(),
            types.ChatMemberStatusBanned().getType(),
            types.ChatMemberStatusRestricted().getType(),
        }:
            if user_status == types.ChatMemberStatusBanned().getType():
                ub = await self.get_client(chat_id)
                if isinstance(ub, types.Error):
                    return ub

                user_id = ub.me.id
                await self.bot.setChatMemberStatus(
                    chat_id=chat_id,
                    member_id=types.MessageSenderUser(user_id),
                    status=types.ChatMemberStatusMember(),
                )

            join = await self._join_ub(chat_id)
            return join if isinstance(join, types.Error) else types.Ok()
        return types.Ok()

    async def _join_ub(self, chat_id: int) -> Union[types.Ok, types.Error]:
        """
        Handles the userbot joining a chat via invite link or approval.
        """
        client = await self.get_client(chat_id)
        if isinstance(client, types.Error):
            return client

        invite_link = chat_invite_cache.get(chat_id)
        if not invite_link:
            get_link = await self.bot.createChatInviteLink(chat_id, name="TgMusicBot")
            if isinstance(get_link, types.Error):
                return get_link
            invite_link = get_link.invite_link

        if not invite_link:
            return types.Error(
                code=400, message=f"Failed to get invite link for chat {chat_id}"
            )

        chat_invite_cache[chat_id] = invite_link
        invite_link = invite_link.replace("https://t.me/+", "https://t.me/joinchat/")
        user_id = client.me.id
        cache_key = f"{chat_id}:{user_id}"
        try:
            await client.join_chat(invite_link)
            user_status_cache[cache_key] = types.ChatMemberStatusMember()
            return types.Ok()
        except errors.InviteRequestSent:
            ok = await self.bot.processChatJoinRequest(
                chat_id=chat_id, user_id=user_id, approve=True
            )
            if isinstance(ok, types.Error):
                return ok
            user_status_cache[cache_key] = types.ChatMemberStatusMember()
            return ok
        except errors.UserAlreadyParticipant:
            user_status_cache[cache_key] = types.ChatMemberStatusMember()
            return types.Ok()
        except errors.InviteHashExpired:
            return types.Error(
                code=400,
                message=f"Invite link has expired or my assistant (<code>{user_id}</code>) is banned from this group.",
            )
        except Exception as e:
            return types.Error(code=400, message=f"Failed to join {user_id}: {e}")


call = Calls()
