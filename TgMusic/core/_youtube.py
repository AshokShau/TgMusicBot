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
    """Utility class for YouTube-related operations."""

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
        """Clean the query by removing unnecessary parameters."""
        return query.split("&")[0].split("#")[0].strip()

    @staticmethod
    def is_valid_url(url: Optional[str]) -> bool:
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
        """Extract video ID from various YouTube URL formats."""
        for pattern in (
            YouTubeUtils.YOUTUBE_VIDEO_PATTERN,
            YouTubeUtils.YOUTUBE_SHORTS_PATTERN,
        ):
            if match := pattern.match(url):
                return match.group(1)
        return None

    @staticmethod
    async def normalize_youtube_url(url: str) -> Optional[str]:
        """Normalize different YouTube URL formats to standard watch URL."""
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
        """Create PlatformTracks object from data."""
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
        """Format track data into a consistent structure."""
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
        """Create TrackInfo from formatted track data."""
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
        """
        Convert duration string (HH:MM:SS or MM:SS) to seconds.

        Args:
            duration: Time string to convert

        Returns:
            int: Duration in seconds
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
        """Get a random cookie file from the 'cookies' directory."""
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
        """
        Download audio using the API.
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
        """Construct yt-dlp parameters based on video/audio requirements."""
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
        """Download YouTube media using yt-dlp.

        Args:
            video_id (str): YouTube video ID.
            video (bool): True to download video; False for audio only.

        Returns:
            Optional[str]: File path of the downloaded media, or None on failure.
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
    """Handles YouTube music data operations including:
    - URL validation
    - Track information retrieval
    - Search functionality
    - Audio/video downloads

    Uses both direct API calls and YouTube Data API for comprehensive coverage.
    """

    def __init__(self, query: Optional[str] = None) -> None:
        """Initialize with optional query (URL or search term).

        Args:
            query: YouTube URL or search term to process
        """
        self.query = YouTubeUtils.clean_query(query) if query else None

    def is_valid(self) -> bool:
        """Validate YouTube URL format.

        Returns:
            bool: True if URL matches YouTube patterns
        """
        return YouTubeUtils.is_valid_url(self.query)

    async def get_info(self) -> Union[PlatformTracks, types.Error]:
        """Retrieve track information from YouTube URL.

        Returns:
            PlatformTracks: Contains track metadata
            types.Error: If URL is invalid or request fails
        """
        if not self.query or not self.is_valid():
            return types.Error(code=400, message="Invalid YouTube URL provided")

        data = await self._fetch_data(self.query)
        if not data:
            return types.Error(code=404, message="Could not retrieve track information")

        return YouTubeUtils.create_platform_tracks(data)

    async def search(self) -> Union[PlatformTracks, types.Error]:
        """Search YouTube for tracks matching the query.

        Returns:
            PlatformTracks: Contains search results
            types.Error: If query is invalid or search fails
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
        """Get detailed track information.

        Returns:
            TrackInfo: Detailed track metadata
            types.Error: If track cannot be found
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
        """Download audio/video track from YouTube.

        Args:
            track: TrackInfo containing download details
            video: Whether to download video (default: False)

        Returns:
            Path: Location of downloaded file
            types.Error: If download fails
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
        """Internal method to fetch YouTube data.

        Handles both videos and playlists.
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
        """Retrieve metadata for a single YouTube video."""
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
        """Retrieve metadata for a YouTube playlist."""
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
