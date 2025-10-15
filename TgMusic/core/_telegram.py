#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Optional, Union

from cachetools import TTLCache
from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config


class Telegram:
    """
    Helper class to validate and process playable Telegram media messages.
    """

    UNSUPPORTED_TYPES = (
        types.MessageText,
        types.MessagePhoto,
        types.MessageSticker,
        types.MessageAnimation,
    )
    MAX_FILE_SIZE = config.MAX_FILE_SIZE
    DownloaderCache = TTLCache(maxsize=5000, ttl=600)

    def __init__(self):
        self._file_info: Optional[tuple[int, str]] = None

    @staticmethod
    def _extract_file_info(content: types.MessageContent) -> tuple[int, str]:
        try:
            if isinstance(content, types.MessageVideo):
                return (
                    content.video.video.size,
                    content.video.file_name or "Video.mp4",
                )
            elif isinstance(content, types.MessageAudio):
                return (
                    content.audio.audio.size,
                    content.audio.file_name or "Audio.mp3",
                )
            elif isinstance(content, types.MessageVoiceNote):
                return content.voice_note.voice.size, "VoiceNote.ogg"
            elif isinstance(content, types.MessageVideoNote):
                return content.video_note.video.size, "VideoNote.mp4"
            elif isinstance(content, types.MessageDocument):
                mime = (content.document.mime_type or "").lower()
                if mime.startswith(("audio/", "video/")):
                    return (
                        content.document.document.size,
                        content.document.file_name or "Document.mp4",
                    )
            return 0, "UnknownMedia"
        except Exception as e:
            LOGGER.error("Error while extracting file info: %s", e)

        LOGGER.info("Unsupported content type: %s", type(content).__name__)
        return 0, "UnknownMedia"

    def is_valid(self, msg: Optional[types.Message]) -> bool:
        if not msg or isinstance(msg, types.Error):
            return False

        content = msg.content
        if isinstance(content, self.UNSUPPORTED_TYPES):
            return False

        file_size, _ = self._extract_file_info(content)
        return 0 < file_size <= self.MAX_FILE_SIZE

    async def download_msg(
        self, dl_msg: types.Message, message: types.Message
    ) -> tuple[Union[types.Error, types.LocalFile], str]:
        if not self.is_valid(dl_msg):
            return (
                types.Error(code=0, message="Invalid or unsupported media file."),
                "InvalidMedia",
            )

        unique_id = dl_msg.remote_unique_file_id
        chat_id = message.chat_id if message else dl_msg.chat_id
        file_size, file_name = self._extract_file_info(dl_msg.content)

        if unique_id not in Telegram.DownloaderCache:
            Telegram.DownloaderCache[unique_id] = {
                "chat_id": chat_id,
                "remote_file_id": dl_msg.remote_file_id,
                "filename": file_name,
                "message_id": message.id,
            }
        return await dl_msg.download(), file_name

    @staticmethod
    def get_cached_metadata(
        unique_id: str,
    ) -> Optional[dict[str, Union[int, str, str, int]]]:
        return Telegram.DownloaderCache.get(unique_id)

    @staticmethod
    def clear_cache(unique_id: str):
        return Telegram.DownloaderCache.pop(unique_id, None)


tg = Telegram()
