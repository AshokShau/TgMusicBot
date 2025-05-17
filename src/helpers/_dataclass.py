#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from pathlib import Path
from typing import Union

from pydantic import BaseModel

class ChannelPlay(BaseModel):
    chat_id: int
    is_channel: bool

class CachedTrack(BaseModel):
    url: str
    name: str
    artist: str
    loop: int
    user: str
    file_path: Union[str, Path]
    thumbnail: str
    track_id: str
    duration: int = 0
    is_video: bool
    platform: str
    channel: ChannelPlay


class TrackInfo(BaseModel):
    url: str
    cdnurl: str
    key: str
    name: str
    artist: str
    album: str
    tc: str
    cover: str
    lyrics: str
    duration: int
    year: int
    platform: str


class MusicTrack(BaseModel):
    url: str
    name: str
    artist: str
    id: str
    year: int
    cover: str
    duration: int
    platform: str


class PlatformTracks(BaseModel):
    tracks: list[MusicTrack]
