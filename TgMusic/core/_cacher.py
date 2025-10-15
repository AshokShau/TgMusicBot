#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from collections import deque
from pathlib import Path
from typing import Any, Optional, TypeAlias, Union

from cachetools import TTLCache
from pytdbot import types

from TgMusic.core._dataclass import CachedTrack

chat_invite_cache = TTLCache(maxsize=1000, ttl=1000)

ChatMemberStatus: TypeAlias = Union[
    types.ChatMemberStatusCreator,
    types.ChatMemberStatusAdministrator,
    types.ChatMemberStatusMember,
    types.ChatMemberStatusRestricted,
    types.ChatMemberStatusLeft,
    types.ChatMemberStatusBanned,
]

ChatMemberStatusResult: TypeAlias = Union[ChatMemberStatus, types.Error]
user_status_cache: TTLCache[str, ChatMemberStatus] = TTLCache(maxsize=5000, ttl=1000)


class ChatCacher:
    """Manages in-memory caching for chat-related data, like song queues.

    This class provides an interface to manage song queues for different chats,
    including adding, removing, and retrieving tracks, as well as handling
    the active state of chat music sessions.

    Attributes:
        chat_cache (dict[int, dict[str, Any]]): A dictionary holding the
            cache for each chat, with chat IDs as keys. Each chat's cache
            contains its song queue and activity status.
    """

    def __init__(self):
        """Initializes the ChatCacher with an empty cache."""
        self.chat_cache: dict[int, dict[str, Any]] = {}

    def add_song(self, chat_id: int, song: CachedTrack) -> CachedTrack:
        """Adds a song to the queue of a specific chat.

        If the chat is not already cached, it is initialized with an active
        status and an empty queue.

        Args:
            chat_id (int): The ID of the chat.
            song (CachedTrack): The song to add to the queue.

        Returns:
            CachedTrack: The song that was added.
        """
        data = self.chat_cache.setdefault(
            chat_id, {"is_active": True, "queue": deque()}
        )
        data["queue"].append(song)
        return song

    def get_upcoming_track(self, chat_id: int) -> Optional[CachedTrack]:
        """Retrieves the next song in the queue for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            Optional[CachedTrack]: The upcoming song, or None if the queue is
                empty or has only one song.
        """
        queue = self.chat_cache.get(chat_id, {}).get("queue")
        return queue[1] if queue and len(queue) > 1 else None

    def get_playing_track(self, chat_id: int) -> Optional[CachedTrack]:
        """Retrieves the currently playing song for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            Optional[CachedTrack]: The current song, or None if the queue is empty.
        """
        queue = self.chat_cache.get(chat_id, {}).get("queue")
        return queue[0] if queue else None

    def remove_current_song(
        self, chat_id: int, disk_clear: bool = True
    ) -> Optional[CachedTrack]:
        """Removes the current song from a chat's queue.

        This also handles the deletion of the associated audio file and
        thumbnail from the disk if `disk_clear` is True.

        Args:
            chat_id (int): The ID of the chat.
            disk_clear (bool): If True, deletes the song's files from disk.
                Defaults to True.

        Returns:
            Optional[CachedTrack]: The song that was removed, or None if the
                queue was empty.
        """
        queue = self.chat_cache.get(chat_id, {}).get("queue")
        if not queue:
            return None

        removed = queue.popleft()
        if disk_clear and getattr(removed, "file_path", None):
            try:
                file_path = (
                    Path(removed.file_path)
                    if isinstance(removed.file_path, str)
                    else removed.file_path
                )
                file_path.unlink(missing_ok=True)
                thumb_path = Path(f"database/photos/{removed.track_id}.png")
                thumb_path.unlink(missing_ok=True)
            except OSError:
                pass
        return removed

    def is_active(self, chat_id: int) -> bool:
        """Checks if the music session for a chat is active.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            bool: True if the session is active, False otherwise.
        """
        return self.chat_cache.get(chat_id, {}).get("is_active", False)

    def set_active(self, chat_id: int, active: bool):
        """Sets the active status for a chat's music session.

        Args:
            chat_id (int): The ID of the chat.
            active (bool): The new active status to set.
        """
        data = self.chat_cache.setdefault(
            chat_id, {"is_active": active, "queue": deque()}
        )
        data["is_active"] = active

    def clear_chat(self, chat_id: int, disk_clear: bool = True):
        """Clears all cached data for a chat, including the queue.

        If `disk_clear` is True, it also deletes all associated song files
        from the disk.

        Args:
            chat_id (int): The ID of the chat to clear.
            disk_clear (bool): If True, deletes song files from disk.
                Defaults to True.
        """
        if disk_clear and chat_id in self.chat_cache:
            queue = self.chat_cache[chat_id].get("queue", deque())
            for track in queue:
                if track.file_path:
                    try:
                        file_path = (
                            Path(track.file_path)
                            if isinstance(track.file_path, str)
                            else track.file_path
                        )
                        file_path.unlink(missing_ok=True)
                    except (OSError, TypeError, AttributeError, KeyError):
                        pass
        self.chat_cache.pop(chat_id, None)

    def get_queue_length(self, chat_id: int) -> int:
        """Gets the number of songs in a chat's queue.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            int: The length of the queue.
        """
        return len(self.chat_cache.get(chat_id, {}).get("queue", deque()))

    def get_loop_count(self, chat_id: int) -> int:
        """Gets the loop count for the currently playing song in a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            int: The loop count, or 0 if no song is playing.
        """
        queue = self.chat_cache.get(chat_id, {}).get("queue", deque())
        return queue[0].loop if queue else 0

    def set_loop_count(self, chat_id: int, loop: int) -> bool:
        """Sets the loop count for the currently playing song.

        Args:
            chat_id (int): The ID of the chat.
            loop (int): The number of times to loop the song.

        Returns:
            bool: True if the loop count was set, False if no song is playing.
        """
        if queue := self.chat_cache.get(chat_id, {}).get("queue", deque()):
            queue[0].loop = loop
            return True
        return False

    def remove_track(self, chat_id: int, queue_index: int) -> bool:
        """Removes a specific track from the queue by its index.

        Args:
            chat_id (int): The ID of the chat.
            queue_index (int): The index of the track to remove.

        Returns:
            bool: True if the track was removed, False if the index was invalid.
        """
        queue = self.chat_cache.get(chat_id, {}).get("queue")
        if queue and 0 <= queue_index < len(queue):
            queue_list = list(queue)
            queue_list.pop(queue_index)
            self.chat_cache[chat_id]["queue"] = deque(queue_list)
            return True
        return False

    def get_queue(self, chat_id: int) -> list[CachedTrack]:
        """Retrieves the entire song queue for a chat as a list.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            list[CachedTrack]: A list of the songs in the queue.
        """
        return list(self.chat_cache.get(chat_id, {}).get("queue", deque()))

    def get_active_chats(self) -> list[int]:
        """Gets a list of all chat IDs with an active music session.

        Returns:
            list[int]: A list of active chat IDs.
        """
        return [
            chat_id for chat_id, data in self.chat_cache.items() if data["is_active"]
        ]


chat_cache: ChatCacher = ChatCacher()
