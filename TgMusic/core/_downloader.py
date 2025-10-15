# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

from abc import ABC, abstractmethod
from pathlib import Path
from typing import Optional, Union

from pytdbot import types

from ._config import config
from ._dataclass import PlatformTracks, TrackInfo


class MusicService(ABC):
    """Abstract base class for all music service integrations.

    This class defines the standard interface that all music service handlers
    (like YouTube, Spotify, etc.) must implement. It ensures that each service
    provides a consistent set of methods for validation, searching, and
    downloading tracks.
    """

    @abstractmethod
    def is_valid(self) -> bool:
        """Checks if the provided query is a valid URL for the service."""
        ...

    @abstractmethod
    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieves information for a playlist, album, or track from a URL."""
        ...

    @abstractmethod
    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Searches for tracks using a query string."""
        ...

    @abstractmethod
    async def get_track(self) -> Union[TrackInfo, types.Error]:
        """Gets detailed information for a single track."""
        ...

    @abstractmethod
    async def download_track(
        self, track_info: TrackInfo, video: bool = False
    ) -> Union[Path, types.Error]:
        """Downloads a track to local storage.

        Args:
            track_info (TrackInfo): The track's metadata.
            video (bool): Flag to download video instead of audio. Defaults to False.

        Returns:
            Union[Path, types.Error]: The path to the downloaded file or an Error.
        """
        ...


class DownloaderWrapper(MusicService):
    """A wrapper that selects and uses the appropriate music service.

    This class acts as a factory and a facade for the different music service
    implementations. It determines the correct service to use based on the
    input query (e.g., if it's a YouTube URL, it uses the YouTube service).
    If the query is not a specific URL, it falls back to a default service
    for searching.

    Attributes:
        query (Optional[str]): The input search term or URL.
        service (MusicService): The music service instance chosen to handle the query.
    """

    def __init__(self, query: Optional[str] = None) -> None:
        """Initializes the DownloaderWrapper.

        Args:
            query (Optional[str]): The search term or URL. Defaults to None.
        """
        self.query = query
        self.service = self._get_service()

    def _get_service(self) -> MusicService:
        """Determines and returns the appropriate music service instance.

        It first checks if the query matches the URL pattern of any available
        service. If not, it uses the default service specified in the bot's
        configuration.

        Returns:
            MusicService: An instance of the selected music service handler.
        """
        from ._api import ApiData
        from ._jiosaavn import JiosaavnData
        from ._youtube import YouTubeData

        services = [YouTubeData, JiosaavnData, ApiData]
        if service := next(
            (s(self.query) for s in services if s(self.query).is_valid()), None
        ):
            return service

        fallback = {
            "youtube": YouTubeData,
            "spotify": ApiData,
            "jiosaavn": JiosaavnData,
        }.get(config.DEFAULT_SERVICE, YouTubeData)

        return (
            ApiData(self.query)
            if fallback == ApiData and config.API_URL and config.API_KEY
            else fallback(self.query)
        )

    def is_valid(self) -> bool:
        """Delegates the validity check to the selected service.

        Returns:
            bool: True if the query is a valid URL for the service, False otherwise.
        """
        return self.service.is_valid()

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Delegates the get_info call to the selected service.

        Returns:
            Union[PlatformTracks, types.Error]: Track information or an error.
        """
        return await self.service.get_info()

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Delegates the search call to the selected service.

        Returns:
            Union[PlatformTracks, types.Error]: Search results or an error.
        """
        return await self.service.search()

    async def get_track(self) -> Union[TrackInfo, types.Error]:
        """Delegates the get_track call to the selected service.

        Returns:
            Union[TrackInfo, types.Error]: Detailed track info or an error.
        """
        return await self.service.get_track()

    async def download_track(
        self, track_info: TrackInfo, video: bool = False
    ) -> Union[Path, types.Error]:
        """Delegates the download_track call to the selected service.

        Args:
            track_info (TrackInfo): The track's metadata.
            video (bool): Flag to download video instead of audio. Defaults to False.

        Returns:
            Union[Path, types.Error]: The path to the downloaded file or an Error.
        """
        return await self.service.download_track(track_info, video)
