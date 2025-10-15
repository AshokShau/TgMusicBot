#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import re
from pathlib import Path
from typing import Optional, Union

from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config
from ._dataclass import MusicTrack, PlatformTracks, TrackInfo
from ._downloader import MusicService
from ._httpx import HttpxClient
from ._spotify_dl_helper import SpotifyDownload


class ApiData(MusicService):
    """API integration handler for multiple music streaming platforms.

    Provides functionality to:
    - Validate and process music URLs
    - Retrieve track information
    - Search across platforms
    - Download tracks
    """

    # Platform URL validation patterns
    URL_PATTERNS = {
        "apple_music": re.compile(
            r"^(https?://)?([a-z0-9-]+\.)*music\.apple\.com/"
            r"([a-z]{2}/)?"
            r"(album|playlist|song)/[a-zA-Z0-9\-._]+/(pl\.[a-zA-Z0-9]+|\d+)(\?.*)?$",
            re.IGNORECASE,
        ),
        "spotify": re.compile(
            r"^(https?://)?([a-z0-9-]+\.)*spotify\.com/"
            r"(track|playlist|album|artist)/[a-zA-Z0-9]+(\?.*)?$",
            re.IGNORECASE,
        ),
        "soundcloud": re.compile(
            r"^(https?://)?([a-z0-9-]+\.)*soundcloud\.com/"
            r"[a-zA-Z0-9_-]+(/(sets)?/[a-zA-Z0-9_-]+)?(\?.*)?$",
            re.IGNORECASE,
        ),
    }

    def __init__(self, query: Optional[str] = None) -> None:
        """Initialize the API handler with optional query.

        Args:
            query: URL or search term to process
        """
        self.query = self._sanitize_query(query) if query else None
        self.api_url = config.API_URL.rstrip("/") if config.API_URL else None
        self.api_key = config.API_KEY
        self.client = HttpxClient()

    @staticmethod
    def _sanitize_query(query: str) -> str:
        """Clean and standardize input queries.

        Removes:
        - URL fragments (#)
        - Query parameters (?)
        - Leading/trailing whitespace
        """
        return query.strip().split("?")[0].split("#")[0]

    def is_valid(self) -> bool:
        """Validate if URL matches supported platform patterns.

        Returns:
            bool: True if URL matches any platform pattern
        """
        if not all([self.query, self.api_key, self.api_url]):
            return False

        return any(pattern.match(self.query) for pattern in self.URL_PATTERNS.values())

    async def _make_api_request(
        self, endpoint: str, params: Optional[dict] = None
    ) -> Optional[dict]:
        request_url = f"{self.api_url}/{endpoint.lstrip('/')}"
        return await self.client.make_request(request_url, params=params)

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieve track information from a valid URL.

        Returns:
            PlatformTracks: Contains track metadata
            types.Error: If URL is invalid or request fails
        """
        if not self.query or not self.is_valid():
            return types.Error(400, "Invalid or unsupported URL provided")

        response = await self._make_api_request("get_url", {"url": self.query})
        return self._parse_tracks_response(response) or types.Error(
            404, "No track information found"
        )

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Search for tracks across supported platforms.

        Returns:
            PlatformTracks: Contains search results
            types.Error: If query is invalid or search fails
        """
        if not self.query:
            return types.Error(400, "No search query provided")

        # If query is a valid URL, get info directly
        if self.is_valid():
            return await self.get_info()

        response = await self._make_api_request("search", {"query": self.query})
        return self._parse_tracks_response(response) or types.Error(
            404, "No results found for search query"
        )

    async def get_track(self) -> Union[TrackInfo, types.Error]:
        """Get detailed track information.

        Returns:
            TrackInfo: Detailed track metadata
            types.Error: If track cannot be found
        """
        if not self.query:
            return types.Error(400, "No track identifier provided")

        response = await self._make_api_request("track", {"url": self.query})
        return (
            TrackInfo(**response) if response else types.Error(404, "Track not found")
        )

    async def download_track(
        self, track: TrackInfo, video: bool = False
    ) -> Union[Path, types.Error]:
        """Download a track to local storage.

        Args:
            track: TrackInfo object containing download details
            video: Whether to download video (default: False)

        Returns:
            Path: Location of downloaded file
            types.Error: If download fails
        """
        if not track:
            return types.Error(400, "Invalid track information provided")

        # Handle platform-specific download methods
        if track.platform.lower() == "spotify":
            spotify_result = await SpotifyDownload(track).process()
            if isinstance(spotify_result, types.Error):
                LOGGER.error(f"Spotify download failed: {spotify_result.message}")
            return spotify_result

        # if track.platform.lower() == "youtube":
        #     return await YouTubeData().download_track(track, video)

        if not track.cdnurl:
            error_msg = f"No download URL available for track: {track.tc}"
            LOGGER.error(error_msg)
            return types.Error(400, error_msg)

        # Standard download handling
        download_path = config.DOWNLOADS_DIR / f"{track.tc}.mp3"
        download_result = await self.client.download_file(track.cdnurl, download_path)

        if not download_result.success:
            LOGGER.warning(
                f"Download failed for track {track.tc}: {download_result.error}"
            )
            return types.Error(
                500, f"Download failed: {download_result.error or track.tc}"
            )

        return download_result.file_path

    @staticmethod
    def _parse_tracks_response(
        response_data: Optional[dict],
    ) -> Union[PlatformTracks, types.Error]:
        """Parse and validate API response data.

        Args:
            response_data: Raw API response

        Returns:
            PlatformTracks: Validated track data
            types.Error: If response is invalid
        """
        if not response_data or "results" not in response_data:
            return types.Error(404, "Invalid API response format")

        try:
            tracks = [
                MusicTrack(**track_data)
                for track_data in response_data["results"]
                if isinstance(track_data, dict)
            ]
            return (
                PlatformTracks(tracks=tracks)
                if tracks
                else types.Error(404, "No valid tracks found")
            )
        except Exception as parse_error:
            LOGGER.error(f"Failed to parse tracks: {parse_error}")
            return types.Error(500, "Failed to process track data")
