#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pathlib import Path
from typing import Union

from pydantic import BaseModel


class CachedTrack(BaseModel):
    """Represents a track that is cached for playback.

    Attributes:
        url (str): The URL of the track.
        name (str): The name of the track.
        loop (int): The number of times the track should be looped.
        user (str): The user who requested the track.
        file_path (Union[str, Path]): The local file path of the track.
        thumbnail (str): The URL or local path of the track's thumbnail.
        track_id (str): A unique identifier for the track.
        duration (int): The duration of the track in seconds.
        is_video (bool): A flag indicating if the track is a video.
        platform (str): The platform from which the track originated.
    """

    url: str
    name: str
    loop: int
    user: str
    file_path: Union[str, Path]
    thumbnail: str
    track_id: str
    duration: int = 0
    is_video: bool
    platform: str


class TrackInfo(BaseModel):
    """Represents detailed information about a specific track.

    Attributes:
        url (str): The original URL of the track.
        cdnurl (str): The CDN URL for downloading the track.
        key (str): A key associated with the track.
        name (str): The name of the track.
        tc (str): A track code or identifier.
        cover (str): The URL of the track's cover art.
        duration (int): The duration of the track in seconds.
        platform (str): The platform from which the track information was fetched.
    """

    url: str
    cdnurl: str
    key: str
    name: str
    tc: str
    cover: str
    duration: int
    platform: str


class MusicTrack(BaseModel):
    """Represents a single music track from a platform's search results or playlist.

    Attributes:
        url (str): The URL of the track.
        name (str): The name of the track.
        id (str): The unique identifier for the track on its platform.
        cover (str): The URL of the track's cover art.
        duration (int): The duration of the track in seconds.
        platform (str): The name of the music platform.
    """

    url: str
    name: str
    id: str
    cover: str
    duration: int
    platform: str


class PlatformTracks(BaseModel):
    """Represents a collection of tracks from a music platform.

    Attributes:
        tracks (list[MusicTrack]): A list of `MusicTrack` objects.
    """

    tracks: list[MusicTrack]
