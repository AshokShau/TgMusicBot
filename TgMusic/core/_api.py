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
    """Handles interactions with music streaming platform APIs.

    This class provides a unified interface to validate and process music URLs,
    retrieve track information, perform searches across different platforms,
    and download tracks. It supports platforms like Apple Music, Spotify, and
    SoundCloud.

    Attributes:
        URL_PATTERNS (dict): A dictionary of regular expression patterns for
            validating URLs from different platforms.
        query (Optional[str]): The URL or search term to be processed.
        api_url (Optional[str]): The base URL for the external API.
        api_key (Optional[str]): The API key for authenticating with the external API.
        client (HttpxClient): An HTTP client for making web requests.
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
        """Initializes the ApiData instance.

        Args:
            query (Optional[str]): The URL or search term to process.
                Defaults to None.
        """
        self.query = self._sanitize_query(query) if query else None
        self.api_url = config.API_URL.rstrip("/") if config.API_URL else None
        self.api_key = config.API_KEY
        self.client = HttpxClient()

    @staticmethod
    def _sanitize_query(query: str) -> str:
        """Cleans and standardizes an input query string.

        This method removes URL fragments, query parameters, and any
        leading or trailing whitespace to produce a clean base URL or
        search term.

        Args:
            query (str): The input string to sanitize.

        Returns:
            str: The sanitized query string.
        """
        return query.strip().split("?")[0].split("#")[0]

    def is_valid(self) -> bool:
        """Validates if the query is a URL matching supported platform patterns.

        This method checks if the instance's query attribute is a valid and
        supported URL by comparing it against the predefined regex patterns.
        It also ensures that the API URL and key are configured.

        Returns:
            bool: True if the query is a valid URL for a supported platform,
                False otherwise.
        """
        if not all([self.query, self.api_key, self.api_url]):
            return False

        return any(pattern.match(self.query) for pattern in self.URL_PATTERNS.values())

    async def _make_api_request(
        self, endpoint: str, params: Optional[dict] = None
    ) -> Optional[dict]:
        """Makes a request to the external music API.

        Args:
            endpoint (str): The API endpoint to call.
            params (Optional[dict]): A dictionary of parameters to include in
                the request. Defaults to None.

        Returns:
            Optional[dict]: The JSON response from the API as a dictionary,
                or None if the request fails.
        """
        request_url = f"{self.api_url}/{endpoint.lstrip('/')}"
        return await self.client.make_request(request_url, params=params)

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieves track information for a given URL.

        This method validates the URL and then queries the API to get
        metadata for the track, playlist, or album.

        Returns:
            Union[PlatformTracks, types.Error]: An object containing track
                metadata, or an Error object if the URL is invalid or the
                API request fails.
        """
        if not self.query or not self.is_valid():
            return types.Error(400, "Invalid or unsupported URL provided")

        response = await self._make_api_request("get_url", {"url": self.query})
        return self._parse_tracks_response(response) or types.Error(
            404, "No track information found"
        )

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Searches for tracks across supported platforms using a query.

        If the query is a valid URL, it fetches information directly. Otherwise,
        it performs a search using the provided query term.

        Returns:
            Union[PlatformTracks, types.Error]: An object containing search
                results, or an Error object if the query is empty or the
                search fails.
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
        """Retrieves detailed information for a single track.

        Returns:
            Union[TrackInfo, types.Error]: An object with detailed track
                metadata, or an Error object if the track cannot be found.
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
        """Downloads a track to local storage.

        This method handles the download process for a given track. It can
        use platform-specific downloaders (e.g., for Spotify) or a standard
        downloader for direct URLs.

        Args:
            track (TrackInfo): An object containing the track's metadata and
                download details.
            video (bool): Whether to download the video version of the track,
                if available. Defaults to False.

        Returns:
            Union[Path, types.Error]: The path to the downloaded file, or an
                Error object if the download fails.
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
        """Parses and validates the API response for track data.

        This method processes the raw dictionary from an API response,
        validates its structure, and converts the track data into a
        `PlatformTracks` object.

        Args:
            response_data (Optional[dict]): The raw data from the API response.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object
                containing the parsed track data, or an `Error` object if the
                response is invalid or cannot be parsed.
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
