#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import re
from pathlib import Path
from typing import Any, Optional, Union

import yt_dlp
from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config
from ._dataclass import MusicTrack, PlatformTracks, TrackInfo
from ._downloader import MusicService
from ._httpx import HttpxClient


class JiosaavnData(MusicService):
    """Handles interactions with the JioSaavn music service.

    This class implements the `MusicService` interface to provide functionality
    for validating JioSaavn URLs, searching for tracks, retrieving track and
    playlist information, and downloading audio. It uses a combination of
    direct API calls to JioSaavn's internal API and `yt-dlp` for robust
    data extraction.

    Attributes:
        JIOSAAVN_SONG_PATTERN (re.Pattern): Regex to validate JioSaavn song URLs.
        JIOSAAVN_PLAYLIST_PATTERN (re.Pattern): Regex to validate JioSaavn
            playlist URLs.
        API_SEARCH_ENDPOINT (str): The URL for JioSaavn's autocomplete API.
        DEFAULT_DURATION (int): A default duration to use if metadata is missing.
    """

    # URL patterns for validation
    JIOSAAVN_SONG_PATTERN = re.compile(
        r"^(https?://)?(www\.)?jiosaavn\.com/song/[\w-]+/[a-zA-Z0-9_-]+", re.IGNORECASE
    )
    JIOSAAVN_PLAYLIST_PATTERN = re.compile(
        r"^(https?://)?(www\.)?jiosaavn\.com/featured/[\w-]+/[a-zA-Z0-9_-]+$",
        re.IGNORECASE,
    )

    # API configuration
    API_SEARCH_ENDPOINT = (
        "https://www.jiosaavn.com/api.php?"
        "__call=autocomplete.get&"
        "query={query}&"
        "_format=json&"
        "_marker=0&"
        "ctx=wap6dot0"
    )

    # Default values for missing metadata
    DEFAULT_DURATION = 0  # seconds

    def __init__(self, query: Optional[str] = None) -> None:
        """Initializes the JiosaavnData handler.

        Args:
            query (Optional[str]): The JioSaavn URL or search term to process.
                Defaults to None.
        """
        self.query = query
        self._ydl_opts = {
            "quiet": True,
            "no_warnings": True,
            "extract_flat": "in_playlist",
            "socket_timeout": 10,
        }

    def is_valid(self) -> bool:
        """Validates if the query is a recognizable JioSaavn URL.

        Returns:
            bool: True if the query matches a JioSaavn song or playlist URL
                pattern, False otherwise.
        """
        if not self.query:
            return False
        return bool(
            self.JIOSAAVN_SONG_PATTERN.match(self.query)
            or self.JIOSAAVN_PLAYLIST_PATTERN.match(self.query)
        )

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Searches JioSaavn for tracks.

        If the query is a valid JioSaavn URL, it retrieves the info directly.
        Otherwise, it uses the JioSaavn API to search for tracks matching
        the query string.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object with
                the search results, or an `Error` object if the search fails.
        """
        if not self.query:
            return types.Error(code=400, message="Search query cannot be empty")

        # Handle direct URL searches
        if self.is_valid():
            return await self.get_info()

        try:
            # Make API request to JioSaavn search endpoint
            response = await HttpxClient().make_request(
                self.API_SEARCH_ENDPOINT.format(query=self.query)
            )

            if not response or not response.get("songs", {}).get("data"):
                return types.Error(
                    code=404, message=f"No results found for: {self.query}"
                )

            # Format and return results
            formatted_tracks = [
                self._format_track(track)
                for track in response["songs"]["data"]
                if track
            ]
            return PlatformTracks(
                tracks=[MusicTrack(**track) for track in formatted_tracks]
            )

        except Exception as error:
            LOGGER.error(f"JioSaavn search failed for '{self.query}': {error}")
            return types.Error(code=500, message=f"Search failed: {str(error)}")

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieves track or playlist information from a JioSaavn URL.

        This method determines whether the URL is for a single song or a
        playlist and fetches the corresponding data.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object
                containing the track metadata, or an `Error` object if the
                URL is invalid or the request fails.
        """
        if not self.query or not self.is_valid():
            return types.Error(code=400, message="Invalid JioSaavn URL provided")

        try:
            if self.JIOSAAVN_SONG_PATTERN.match(self.query):
                data = await self.get_track_data(self.query)
            else:
                data = await self.get_playlist_data(self.query)

            if not data:
                return types.Error(
                    code=404, message="No data found for the provided URL"
                )

            return self._create_platform_tracks(data)
        except Exception as error:
            LOGGER.error(f"Failed to get info for {self.query}: {error}")
            return types.Error(code=500, message="Failed to retrieve track information")

    async def get_track(self) -> Union[TrackInfo, types.Error]:
        """Gets detailed information for a single JioSaavn track.

        This includes metadata required for playback, such as the download URL.

        Returns:
            Union[TrackInfo, types.Error]: A `TrackInfo` object with detailed
                track metadata, or an `Error` object if the track cannot be found.
        """
        if not self.query:
            return types.Error(code=400, message="No track identifier provided")

        # Normalize URL format
        url = self.query if self.is_valid() else self.format_jiosaavn_url(self.query)

        data = await self.get_track_data(url)
        if not data or not data.get("results"):
            return types.Error(code=404, message="Could not retrieve track details")

        track_data = data["results"][0]
        return TrackInfo(
            cdnurl=track_data.get("cdnurl", ""),
            key="nil",
            name=track_data.get("name", ""),
            tc=track_data.get("id", ""),
            cover=track_data.get("cover", ""),
            duration=track_data.get("duration", self.DEFAULT_DURATION),
            url=track_data.get("url", ""),
            platform="jiosaavn",
        )

    async def get_track_data(self, url: str) -> Optional[dict[str, Any]]:
        """Retrieves metadata for a single JioSaavn track using yt-dlp.

        Args:
            url (str): The URL of the JioSaavn track.

        Returns:
            Optional[dict[str, Any]]: A dictionary containing the parsed
                track metadata, or None if the operation fails.
        """
        try:
            with yt_dlp.YoutubeDL(self._ydl_opts) as ydl:
                info = await asyncio.to_thread(ydl.extract_info, url, download=False)
                return {"results": [self._format_track(info)]} if info else None
        except yt_dlp.DownloadError as error:
            LOGGER.error(f"Download error for track {url}: {error}")
        except Exception as error:
            LOGGER.error(f"Unexpected error processing track {url}: {error}")
        return None

    async def get_playlist_data(self, url: str) -> Optional[dict[str, Any]]:
        """Retrieves metadata for a JioSaavn playlist using yt-dlp.

        Args:
            url (str): The URL of the JioSaavn playlist.

        Returns:
            Optional[dict[str, Any]]: A dictionary containing a list of
                parsed tracks from the playlist, or None if the operation fails.
        """
        try:
            with yt_dlp.YoutubeDL(self._ydl_opts) as ydl:
                info = await asyncio.to_thread(ydl.extract_info, url, download=False)

                if not info or not info.get("entries"):
                    LOGGER.warning(f"Empty playlist response for {url}")
                    return None

                return {
                    "results": [
                        self._format_track(track) for track in info["entries"] if track
                    ]
                }
        except yt_dlp.DownloadError as error:
            LOGGER.error(f"Download error for playlist {url}: {error}")
        except Exception as error:
            LOGGER.error(f"Unexpected error processing playlist {url}: {error}")
        return None

    async def download_track(
        self, track: TrackInfo, video: bool = False
    ) -> Union[Path, types.Error]:
        """Downloads an audio track from JioSaavn.

        Note: The `video` parameter is ignored as JioSaavn is an audio-only service.

        Args:
            track (TrackInfo): An object containing the track's metadata and
                download URL.
            video (bool): This parameter is not used. Defaults to False.

        Returns:
            Union[Path, types.Error]: The path to the downloaded file, or an
                `Error` object if the download fails.
        """
        if not track or not track.cdnurl:
            return types.Error(
                code=400, message=f"No download URL available for track: {track.tc}"
            )

        download_path = config.DOWNLOADS_DIR / f"{track.tc}.m4a"
        result = await HttpxClient(max_redirects=1).download_file(
            track.cdnurl, download_path
        )

        if not result.success:
            error_msg = result.error or f"Download failed for track: {track.tc}"
            LOGGER.error(error_msg)
            return types.Error(code=500, message=error_msg)

        return result.file_path

    @staticmethod
    def format_jiosaavn_url(name_and_id: str) -> str:
        """Formats a JioSaavn URL from a combined name and ID string.

        This is used to construct a full URL when only partial information
        is available.

        Args:
            name_and_id (str): A string in the format "title/track_id".

        Returns:
            str: A full, well-formed JioSaavn song URL, or an empty string
                if the input format is invalid.
        """
        if not name_and_id:
            return ""

        try:
            title, song_id = name_and_id.rsplit("/", 1)
            # Clean and format title for URL
            title = re.sub(r'[\(\)"\',]', "", title.lower())
            title = re.sub(r"\s+", "-", title.strip())
            return f"https://www.jiosaavn.com/song/{title}/{song_id}"
        except ValueError:
            LOGGER.warning(f"Invalid name_and_id format: {name_and_id}")
            return ""

    @classmethod
    def _format_track(cls, track_data: dict[str, Any]) -> dict[str, Any]:
        """Formats raw track data into a standardized dictionary structure.

        This method extracts and standardizes key information from the raw
        metadata provided by the JioSaavn API or yt-dlp.

        Args:
            track_data (dict[str, Any]): The raw track metadata.

        Returns:
            dict[str, Any]: A dictionary with standardized keys and values
                representing the track.
        """
        if not track_data:
            return {}

        # Get best available audio format
        formats = track_data.get("formats", [])
        best_format = max(formats, key=lambda x: x.get("abr", 0), default={})
        # Generate display ID from title and URL
        title = track_data.get("title", "")
        url_parts = track_data.get("url", "").split("/")
        display_id = f"{title}/{url_parts[-1]}" if url_parts else title

        return {
            "id": track_data.get("display_id", display_id),
            "tc": track_data.get("display_id", display_id),
            "name": title,
            "duration": track_data.get("duration", cls.DEFAULT_DURATION),
            "cover": track_data.get("thumbnail", ""),
            "platform": "jiosaavn",
            "url": track_data.get("webpage_url", ""),
            "cdnurl": best_format.get("url", ""),
        }

    @staticmethod
    def _create_platform_tracks(
        data: dict[str, Any],
    ) -> Union[PlatformTracks, types.Error]:
        """Creates a PlatformTracks object from raw API data.

        This helper method validates and converts the results from a data
        fetching operation into a `PlatformTracks` object.

        Args:
            data (dict[str, Any]): The raw API response data containing a
                'results' key with a list of tracks.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object, or
                an `Error` if the data is invalid or contains no tracks.
        """
        if not data or not data.get("results"):
            return types.Error(code=404, message="No valid tracks found in response")

        return PlatformTracks(
            tracks=[MusicTrack(**track) for track in data["results"] if track]
        )
