#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Optional, Union

from cachetools import TTLCache
from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config


class Telegram:
    """A helper class for processing and validating playable media from Telegram.

    This class provides methods to check if a Telegram message contains
    supported media (audio or video), extract file information, and manage
    a cache for downloaded media metadata.

    Attributes:
        UNSUPPORTED_TYPES (tuple): A tuple of `pytdbot.types` that are not
            considered playable media.
        MAX_FILE_SIZE (int): The maximum allowed file size for media, sourced
            from the bot's configuration.
        DownloaderCache (TTLCache): A time-to-live cache to store metadata
            of recently processed files.
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
        """Initializes the Telegram helper."""
        self._file_info: Optional[tuple[int, str]] = None

    @staticmethod
    def _extract_file_info(content: types.MessageContent) -> tuple[int, str]:
        """Extracts file size and name from various Telegram message content types.

        Args:
            content (types.MessageContent): The content object from a
                `pytdbot.types.Message`.

        Returns:
            tuple[int, str]: A tuple containing the file size in bytes and the
                filename. Returns (0, "UnknownMedia") for unsupported types.
        """
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
        """Checks if a message contains a valid, playable media file.

        A message is considered valid if it's not an error, contains a
        supported media type, and its file size is within the allowed limit.

        Args:
            msg (Optional[types.Message]): The message to validate.

        Returns:
            bool: True if the message contains valid media, False otherwise.
        """
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
        """Downloads the media from a message and caches its metadata.

        Before downloading, it validates the message using `is_valid`. If valid,
        it proceeds with the download, stores metadata in the cache, and
        returns the download result.

        Args:
            dl_msg (types.Message): The message containing the media to download.
            message (types.Message): The original message that triggered the
                download, used for context (e.g., chat_id).

        Returns:
            tuple[Union[types.Error, types.LocalFile], str]: A tuple containing
                the download result (either a `LocalFile` object or an `Error`)
                and the filename.
        """
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
        """Retrieves cached metadata for a file.

        Args:
            unique_id (str): The remote unique file ID of the media.

        Returns:
            Optional[dict]: The cached metadata dictionary, or None if not found.
        """
        return Telegram.DownloaderCache.get(unique_id)

    @staticmethod
    def clear_cache(unique_id: str):
        """Removes an item from the metadata cache.

        Args:
            unique_id (str): The remote unique file ID to remove from the cache.
        """
        return Telegram.DownloaderCache.pop(unique_id, None)


tg = Telegram()
