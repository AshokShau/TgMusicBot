# Copyright (c) 2025 AshokShau
# Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
# Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import os
import random
import re
from pathlib import Path
from typing import Any, Dict, Optional, Union

from py_yt import Playlist, VideosSearch
from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config
from ._dataclass import MusicTrack, PlatformTracks, TrackInfo
from ._downloader import MusicService
from ._httpx import HttpxClient


class YouTubeUtils:
    """A utility class for handling various YouTube-related operations.

    This class provides a collection of static methods for parsing and
    validating YouTube URLs, formatting track data, and managing downloads
    through different backends like `yt-dlp` and a custom API.

    Attributes:
        YOUTUBE_VIDEO_PATTERN (re.Pattern): Regex for standard YouTube video URLs.
        YOUTUBE_PLAYLIST_PATTERN (re.Pattern): Regex for YouTube playlist URLs.
        YOUTUBE_SHORTS_PATTERN (re.Pattern): Regex for YouTube Shorts URLs.
    """

    # Compile regex patterns once at class level
    YOUTUBE_VIDEO_PATTERN = re.compile(
        r"^(?:https?://)?(?:www\.)?(?:youtube\.com|music\.youtube\.com|youtu\.be)/"
        r"(?:watch\?v=|embed/|v/|shorts/)?([\w-]{11})(?:\?|&|$)",
        re.IGNORECASE,
    )
    YOUTUBE_PLAYLIST_PATTERN = re.compile(
        r"^(?:https?://)?(?:www\.)?(?:youtube\.com|music\.youtube\.com)/"
        r"(?:playlist|watch)\?.*\blist=([\w-]+)",
        re.IGNORECASE,
    )
    YOUTUBE_SHORTS_PATTERN = re.compile(
        r"^(?:https?://)?(?:www\.)?youtube\.com/shorts/([\w-]+)",
        re.IGNORECASE,
    )

    @staticmethod
    def clean_query(query: str) -> str:
        """Cleans a query string by removing unnecessary URL parameters.

        Args:
            query (str): The input query or URL string.

        Returns:
            str: The cleaned string.
        """
        return query.split("&")[0].split("#")[0].strip()

    @staticmethod
    def is_valid_url(url: Optional[str]) -> bool:
        """Checks if a URL is a valid YouTube URL (video, playlist, or short).

        Args:
            url (Optional[str]): The URL to validate.

        Returns:
            bool: True if the URL is a valid YouTube URL, False otherwise.
        """
        if not url:
            return False
        return any(
            pattern.match(url)
            for pattern in (
                YouTubeUtils.YOUTUBE_VIDEO_PATTERN,
                YouTubeUtils.YOUTUBE_PLAYLIST_PATTERN,
                YouTubeUtils.YOUTUBE_SHORTS_PATTERN,
            )
        )

    @staticmethod
    def _extract_video_id(url: str) -> Optional[str]:
        """Extracts the video ID from various YouTube URL formats.

        Args:
            url (str): The YouTube URL.

        Returns:
            Optional[str]: The extracted video ID, or None if not found.
        """
        for pattern in (
            YouTubeUtils.YOUTUBE_VIDEO_PATTERN,
            YouTubeUtils.YOUTUBE_SHORTS_PATTERN,
        ):
            if match := pattern.match(url):
                return match.group(1)
        return None

    @staticmethod
    async def normalize_youtube_url(url: str) -> Optional[str]:
        """Normalizes different YouTube URL formats to the standard watch URL.

        This handles formats like `youtu.be` and `/shorts/`.

        Args:
            url (str): The YouTube URL to normalize.

        Returns:
            Optional[str]: The normalized `youtube.com/watch?v=` URL, or None
                if the input is invalid.
        """
        if not url:
            return None

        # Handle youtu.be short links
        if "youtu.be/" in url:
            video_id = url.split("youtu.be/")[1].partition("?")[0].partition("#")[0]
            return f"https://www.youtube.com/watch?v={video_id}"

        # Handle YouTube shorts
        if "youtube.com/shorts/" in url:
            video_id = url.split("youtube.com/shorts/")[1].split("?")[0]
            return f"https://www.youtube.com/watch?v={video_id}"

        return url

    @staticmethod
    def create_platform_tracks(data: Dict[str, Any]) -> PlatformTracks:
        """Creates a `PlatformTracks` object from a dictionary of data.

        Args:
            data (Dict[str, Any]): A dictionary containing a 'results' key
                with a list of track data.

        Returns:
            PlatformTracks: A `PlatformTracks` object.
        """
        if not data or not data.get("results"):
            return PlatformTracks(tracks=[])

        valid_tracks = [
            MusicTrack(**track)
            for track in data["results"]
            if track and track.get("id")
        ]
        return PlatformTracks(tracks=valid_tracks)

    @staticmethod
    def format_track(track_data: Dict[str, Any]) -> Dict[str, Any]:
        """Formats raw track data into a standardized dictionary structure.

        Args:
            track_data (Dict[str, Any]): The raw track data from an API.

        Returns:
            Dict[str, Any]: A dictionary with standardized keys and values.
        """
        duration = track_data.get("duration", "0:00")
        if isinstance(duration, dict):
            duration = duration.get("secondsText", "0:00")

        # Get the highest quality thumbnail
        cover_url = ""
        if thumbnails := track_data.get("thumbnails"):
            for thumb in reversed(thumbnails):
                if url := thumb.get("url"):
                    cover_url = url
                    break

        return {
            "id": track_data.get("id", ""),
            "name": track_data.get("title", "Unknown Title"),
            "duration": YouTubeUtils.duration_to_seconds(duration),
            "cover": cover_url,
            "year": 0,
            "url": f"https://www.youtube.com/watch?v={track_data.get('id', '')}",
            "platform": "youtube",
        }

    @staticmethod
    async def create_track_info(track_data: dict[str, Any]) -> TrackInfo:
        """Creates a `TrackInfo` object from formatted track data.

        Args:
            track_data (dict[str, Any]): A dictionary of formatted track data.

        Returns:
            TrackInfo: A `TrackInfo` object.
        """
        return TrackInfo(
            cdnurl="None",
            key="None",
            name=track_data.get("name", "Unknown Title"),
            tc=track_data.get("id", ""),
            cover=track_data.get("cover", ""),
            duration=track_data.get("duration", 0),
            platform="youtube",
            url=f"https://youtube.com/watch?v={track_data.get('id', '')}",
        )

    @staticmethod
    def duration_to_seconds(duration: str) -> int:
        """Converts a duration string (HH:MM:SS or MM:SS) to seconds.

        Args:
            duration (str): The time string to convert.

        Returns:
            int: The duration in seconds.
        """
        if not duration:
            return 0

        try:
            parts = list(map(int, duration.split(":")))
            if len(parts) == 3:  # HH:MM:SS
                return parts[0] * 3600 + parts[1] * 60 + parts[2]
            return parts[0] * 60 + parts[1] if len(parts) == 2 else parts[0]
        except (ValueError, AttributeError):
            return 0

    @staticmethod
    async def get_cookie_file() -> Optional[str]:
        """Gets the path of a random cookie file from the cookies directory.

        This is used to bypass YouTube's throttling for `yt-dlp`.

        Returns:
            Optional[str]: The full path to a randomly selected cookie file,
                or None if the directory or files don't exist.
        """
        cookie_dir = "TgMusic/cookies"
        try:
            if not os.path.exists(cookie_dir):
                LOGGER.warning("Cookie directory '%s' does not exist.", cookie_dir)
                return None

            files = await asyncio.to_thread(os.listdir, cookie_dir)
            cookies_files = [f for f in files if f.endswith(".txt")]

            if not cookies_files:
                LOGGER.warning("No cookie files found in '%s'.", cookie_dir)
                return None

            random_file = random.choice(cookies_files)
            return os.path.join(cookie_dir, random_file)
        except Exception as e:
            LOGGER.warning("Error accessing cookie directory: %s", e)
            return None

    @staticmethod
    async def fetch_oembed_data(url: str) -> Optional[dict[str, Any]]:
        """Fetches video metadata using YouTube's oEmbed endpoint.

        This is a lightweight way to get basic video information like title
        and thumbnail.

        Args:
            url (str): The URL of the YouTube video.

        Returns:
            Optional[dict[str, Any]]: A dictionary containing the formatted
                track data, or None on failure.
        """
        oembed_url = f"https://www.youtube.com/oembed?url={url}&format=json"
        data = await HttpxClient().make_request(oembed_url, max_retries=1)
        if data:
            video_id = url.split("v=")[1]
            return {
                "results": [
                    {
                        "id": video_id,
                        "name": data.get("title"),
                        "duration": 0,
                        "artist": data.get("author_name", ""),
                        "cover": data.get("thumbnail_url", ""),
                        "year": 0,
                        "url": f"https://www.youtube.com/watch?v={video_id}",
                        "platform": "youtube",
                    }
                ]
            }
        return None

    @staticmethod
    async def download_with_api(
        video_id: str, is_video: bool = False
    ) -> Union[None, Path]:
        """Downloads a track using the configured external API.

        This method can handle direct downloads or resolve Telegram file links.

        Args:
            video_id (str): The YouTube video ID.
            is_video (bool): Whether to download the video version.
                Defaults to False.

        Returns:
            Union[None, Path]: The path to the downloaded file, or None on failure.
        """
        video_url = f"https://www.youtube.com/watch?v={video_id}"
        httpx = HttpxClient()
        get_track = await httpx.make_request(
            f"{config.API_URL}/track?url={video_url}&video={is_video}"
        )
        if not get_track:
            LOGGER.error("Response from API is empty")
            return None

        track = TrackInfo(**get_track)
        cdnurl = track.cdnurl
        if not cdnurl:
            LOGGER.error("CDN URL not found in response")
            return None

        if not re.fullmatch(r"https:\/\/t\.me\/([a-zA-Z0-9_]{5,})\/(\d+)", cdnurl):
            dl = await httpx.download_file(cdnurl)
            return dl.file_path if dl.success else None

        from TgMusic import client

        info = await client.getMessageLinkInfo(cdnurl)
        if isinstance(info, types.Error) or info.message is None:
            LOGGER.error(f"❌ Could not resolve message from link: {cdnurl}; {info}")
            return None

        msg = await client.getMessage(info.chat_id, info.message.id)
        if isinstance(msg, types.Error):
            LOGGER.error(f"❌ Failed to fetch message with ID {info.message.id}; {msg}")
            return None

        file = await msg.download()
        if isinstance(file, types.Error):
            LOGGER.error(
                f"❌ Failed to download message with ID {info.message.id}; {file}"
            )
            return None
        return Path(file.path)

    @staticmethod
    def _build_ytdlp_params(
        video_id: str, video: bool, cookie_file: Optional[str]
    ) -> list[str]:
        """Constructs the list of command-line arguments for `yt-dlp`.

        Args:
            video_id (str): The YouTube video ID.
            video (bool): True to select video formats, False for audio-only.
            cookie_file (Optional[str]): The path to a cookie file to use.

        Returns:
            list[str]: A list of arguments for the `yt-dlp` command.
        """
        output_template = str(config.DOWNLOADS_DIR / "%(id)s.%(ext)s")

        format_selector = (
            "bestvideo[ext=mp4][height<=1080]+bestaudio[ext=m4a]/best[ext=mp4][height<=1080]"
            if video
            else "bestaudio[ext=m4a]/bestaudio[ext=mp4]/bestaudio[ext=webm]/bestaudio/best"
        )

        ytdlp_params = [
            "yt-dlp",
            "--no-warnings",
            "--quiet",
            "--geo-bypass",
            "--retries",
            "2",
            "--continue",
            "--no-part",
            "--concurrent-fragments",
            "3",
            "--socket-timeout",
            "10",
            "--throttled-rate",
            "100K",
            "--retry-sleep",
            "1",
            "--no-write-thumbnail",
            "--no-write-info-json",
            "--no-embed-metadata",
            "--no-embed-chapters",
            "--no-embed-subs",
            "-o",
            output_template,
            "-f",
            format_selector,
        ]

        if video:
            ytdlp_params += ["--merge-output-format", "mp4"]

        if config.PROXY:
            ytdlp_params += ["--proxy", config.PROXY]
        elif cookie_file:
            ytdlp_params += ["--cookies", cookie_file]

        video_url = f"https://www.youtube.com/watch?v={video_id}"
        ytdlp_params += [video_url, "--print", "after_move:filepath"]

        return ytdlp_params

    @staticmethod
    async def download_with_yt_dlp(video_id: str, video: bool) -> Optional[Path]:
        """Downloads YouTube media using the `yt-dlp` command-line tool.

        This method constructs and executes a `yt-dlp` command to download
        the specified video or audio.

        Args:
            video_id (str): The YouTube video ID to download.
            video (bool): True to download video, False for audio only.

        Returns:
            Optional[Path]: The path to the downloaded file, or None on failure.
        """
        cookie_file = await YouTubeUtils.get_cookie_file()
        ytdlp_params = YouTubeUtils._build_ytdlp_params(video_id, video, cookie_file)

        try:
            LOGGER.debug("Starting yt-dlp download for video ID: %s", video_id)

            proc = await asyncio.create_subprocess_exec(
                *ytdlp_params,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )

            stdout, stderr = await asyncio.wait_for(proc.communicate(), timeout=600)

            if proc.returncode != 0:
                LOGGER.error(
                    "yt-dlp failed for %s (code %d): %s",
                    video_id,
                    proc.returncode,
                    stderr.decode().strip(),
                )
                return None

            downloaded_path_str = stdout.decode().strip()
            if not downloaded_path_str:
                LOGGER.error(
                    "yt-dlp finished but no output path returned for %s", video_id
                )
                return None

            downloaded_path = Path(downloaded_path_str)
            if not downloaded_path.exists():
                LOGGER.error(
                    "yt-dlp reported path but file not found: %s", downloaded_path
                )
                return None

            LOGGER.info("Successfully downloaded %s to %s", video_id, downloaded_path)
            return downloaded_path

        except asyncio.TimeoutError:
            LOGGER.error("yt-dlp timed out for video ID: %s", video_id)
            return None
        except Exception as e:
            LOGGER.error(
                "Unexpected error downloading %s: %r", video_id, e, exc_info=True
            )
            return None


class YouTubeData(MusicService):
    """Implements the `MusicService` interface for YouTube.

    This class provides methods for searching, retrieving information about,
    and downloading tracks and playlists from YouTube.
    """

    def __init__(self, query: Optional[str] = None) -> None:
        """Initializes the YouTubeData handler.

        Args:
            query (Optional[str]): The YouTube URL or search term to process.
                Defaults to None.
        """
        self.query = YouTubeUtils.clean_query(query) if query else None

    def is_valid(self) -> bool:
        """Validates if the query is a recognizable YouTube URL.

        Returns:
            bool: True if the query is a valid YouTube URL, False otherwise.
        """
        return YouTubeUtils.is_valid_url(self.query)

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieves track or playlist information from a YouTube URL.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object
                containing the track metadata, or an `Error` object if the
                URL is invalid or the request fails.
        """
        if not self.query or not self.is_valid():
            return types.Error(code=400, message="Invalid YouTube URL provided")

        data = await self._fetch_data(self.query)
        if not data:
            return types.Error(code=404, message="Could not retrieve track information")

        return YouTubeUtils.create_platform_tracks(data)

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Searches YouTube for videos matching the query.

        If the query is a valid URL, it retrieves the info directly.
        Otherwise, it performs a search using the `py_yt` library.

        Returns:
            Union[PlatformTracks, types.Error]: A `PlatformTracks` object with
                the search results, or an `Error` object if the search fails.
        """
        if not self.query:
            return types.Error(code=400, message="No search query provided")

        # Handle direct URL searches
        if self.is_valid():
            return await self.get_info()

        try:
            search = VideosSearch(self.query, limit=5)
            results = await search.next()

            if not results or not results.get("result"):
                return types.Error(
                    code=404, message=f"No results found for: {self.query}"
                )

            tracks = [
                MusicTrack(**YouTubeUtils.format_track(video))
                for video in results["result"]
            ]
            return PlatformTracks(tracks=tracks)

        except Exception as error:
            LOGGER.error(f"YouTube search failed for '{self.query}': {error}")
            return types.Error(code=500, message=f"Search failed: {str(error)}")

    async def get_track(self) -> Union[TrackInfo, types.Error]:
        """Gets detailed information for a single YouTube video.

        Returns:
            Union[TrackInfo, types.Error]: A `TrackInfo` object with detailed
                track metadata, or an `Error` object if the track cannot be found.
        """
        if not self.query:
            return types.Error(code=400, message="No track identifier provided")

        # Normalize URL/ID format
        url = (
            self.query
            if re.match("^https?://", self.query)
            else f"https://youtube.com/watch?v={self.query}"
        )

        data = await self._fetch_data(url)
        if not data or not data.get("results"):
            return types.Error(code=404, message="Could not retrieve track details")

        return await YouTubeUtils.create_track_info(data["results"][0])

    async def download_track(
        self, track: TrackInfo, video: bool = False
    ) -> Union[Path, types.Error]:
        """Downloads a track from YouTube.

        It first attempts to download using the configured API, and falls back
        to `yt-dlp` if the API method fails or is not configured.

        Args:
            track (TrackInfo): An object containing the track's metadata.
            video (bool): Whether to download the video version. Defaults to False.

        Returns:
            Union[Path, types.Error]: The path to the downloaded file, or an
                `Error` object if the download fails.
        """
        if not track:
            return types.Error(code=400, message="Invalid track information provided")

        # Try API download first if configured
        if config.API_URL and config.API_KEY:
            if api_result := await YouTubeUtils.download_with_api(track.tc, video):
                return api_result

        # Fall back to yt-dlp if API fails or not configured
        dl_path = await YouTubeUtils.download_with_yt_dlp(track.tc, video)
        if not dl_path:
            return types.Error(
                code=500, message="Failed to download track from YouTube"
            )

        return dl_path

    async def _fetch_data(self, url: str) -> Optional[Dict[str, Any]]:
        """Internal helper method to fetch data for a YouTube URL.

        It determines if the URL is a video or playlist and calls the
        appropriate data retrieval method.

        Args:
            url (str): The YouTube URL.

        Returns:
            Optional[Dict[str, Any]]: A dictionary of track data, or None.
        """
        try:
            if YouTubeUtils.YOUTUBE_PLAYLIST_PATTERN.match(url):
                LOGGER.debug(f"Processing YouTube playlist: {url}")
                return await self._get_playlist_data(url)

            LOGGER.debug(f"Processing YouTube video: {url}")
            return await self._get_video_data(url)
        except Exception as error:
            LOGGER.error(f"Data fetch failed for {url}: {error}")
            return None

    @staticmethod
    async def _get_video_data(url: str) -> Optional[Dict[str, Any]]:
        """Retrieves metadata for a single YouTube video.

        It first tries the lightweight oEmbed endpoint and falls back to the
        more comprehensive search API if needed.

        Args:
            url (str): The URL of the video.

        Returns:
            Optional[Dict[str, Any]]: A dictionary of track data, or None.
        """
        normalized_url = await YouTubeUtils.normalize_youtube_url(url)
        if not normalized_url:
            return None

        # Try oEmbed first
        if oembed_data := await YouTubeUtils.fetch_oembed_data(normalized_url):
            return oembed_data

        # Fall back to search API
        try:
            search = VideosSearch(normalized_url, limit=1)
            results = await search.next()

            if not results or not results.get("result"):
                return None

            return {"results": [YouTubeUtils.format_track(results["result"][0])]}
        except Exception as error:
            LOGGER.error(f"Video data fetch failed: {error}")
            return None

    @staticmethod
    async def _get_playlist_data(url: str) -> Optional[Dict[str, Any]]:
        """Retrieves metadata for all videos in a YouTube playlist.

        Args:
            url (str): The URL of the playlist.

        Returns:
            Optional[Dict[str, Any]]: A dictionary containing a list of all
                tracks in the playlist, or None on failure.
        """
        try:
            playlist = await Playlist.getVideos(url)
            if not playlist or not playlist.get("videos"):
                return None

            return {
                "results": [
                    YouTubeUtils.format_track(track)
                    for track in playlist["videos"]
                    if track.get("id")  # Filter valid tracks
                ]
            }
        except Exception as error:
            LOGGER.error(f"Playlist data fetch failed: {error}")
            return None
