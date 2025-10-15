#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import os
import subprocess
from pathlib import Path
from typing import Union

import aiofiles
from Crypto.Cipher import AES
from Crypto.Util import Counter
from pytdbot import types

from TgMusic.logger import LOGGER

from ._config import config
from ._dataclass import TrackInfo
from ._httpx import HttpxClient


async def rebuild_ogg(filename: str) -> None:
    """Manually rebuilds and corrects the headers of a broken OGG file.

    This function is specifically designed to fix OGG files that have been
    decrypted from a raw encrypted stream, which often results in missing or
    incorrect headers. It writes the correct OggS magic number and other
    essential Vorbis header information to the file.

    Args:
        filename (str): The path to the OGG file that needs to be fixed.
    """
    if not os.path.exists(filename):
        LOGGER.error("❌ Error: %s not found.", filename)
        return

    try:
        async with aiofiles.open(filename, "r+b") as ogg_file:
            ogg_s = b"OggS"
            zeroes = b"\x00" * 10
            vorbis_start = b"\x01\x1e\x01vorbis"
            channels = b"\x02"
            sample_rate = b"\x44\xac\x00\x00"
            bit_rate = b"\x00\xe2\x04\x00"
            packet_sizes = b"\xb8\x01"

            await ogg_file.seek(0)
            await ogg_file.write(ogg_s)
            await ogg_file.seek(6)
            await ogg_file.write(zeroes)
            await ogg_file.seek(26)
            await ogg_file.write(vorbis_start)
            await ogg_file.seek(39)
            await ogg_file.write(channels)
            await ogg_file.seek(40)
            await ogg_file.write(sample_rate)
            await ogg_file.seek(48)
            await ogg_file.write(bit_rate)
            await ogg_file.seek(56)
            await ogg_file.write(packet_sizes)
            await ogg_file.seek(58)
            await ogg_file.write(ogg_s)
            await ogg_file.seek(62)
            await ogg_file.write(zeroes)
    except Exception as e:
        LOGGER.error("Error rebuilding OGG file %s: %s", filename, e)


class SpotifyDownload:
    """Handles the download, decryption, and processing of Spotify tracks.

    This class encapsulates the entire workflow for obtaining a playable
    audio file from Spotify's encrypted streams. It manages temporary files
    for the encrypted download, the decrypted raw audio, and the final
    fixed output file.

    Attributes:
        track (TrackInfo): An object containing the track's metadata, including
            the CDN URL and decryption key.
        encrypted_file (str): The path to the temporary encrypted OGG file.
        decrypted_file (str): The path to the temporary decrypted OGG file.
        output_file (str): The path for the final, playable OGG audio file.
    """

    def __init__(self, track: TrackInfo):
        """Initializes the SpotifyDownload helper.

        Args:
            track (TrackInfo): The track information object containing metadata
                like CDN URL, track code (tc), and decryption key.
        """
        self.track = track
        self.encrypted_file = os.path.join(
            config.DOWNLOADS_DIR, f"{track.tc}.encrypted.ogg"
        )
        self.decrypted_file = os.path.join(
            config.DOWNLOADS_DIR, f"{track.tc}.decrypted.ogg"
        )
        self.output_file = os.path.join(config.DOWNLOADS_DIR, f"{track.tc}.ogg")

    async def decrypt_audio(self) -> None:
        """Decrypts the downloaded audio file using AES in CTR mode.

        This method reads the encrypted audio file in chunks, decrypts each
        chunk using the key and IV from the track's metadata, and writes the
        decrypted content to a new file. This stream-based approach is
        memory-efficient.

        Raises:
            Exception: If any error occurs during the decryption process.
        """
        try:
            key = bytes.fromhex(self.track.key)
            iv = bytes.fromhex("72e067fbddcbcf77ebe8bc643f630d93")
            iv_int = int.from_bytes(iv, "big")
            cipher = AES.new(
                key, AES.MODE_CTR, counter=Counter.new(128, initial_value=iv_int)
            )

            chunk_size = 8192  # 8KB chunks
            async with (
                aiofiles.open(self.encrypted_file, "rb") as fin,
                aiofiles.open(self.decrypted_file, "wb") as f,
            ):
                while chunk := await fin.read(chunk_size):
                    decrypted_chunk = cipher.decrypt(chunk)
                    await f.write(decrypted_chunk)
        except Exception as e:
            LOGGER.error("Error decrypting audio file: %s", e)
            raise

    async def fix_audio(self) -> None:
        """Fixes the container and headers of the decrypted audio file using ffmpeg.

        After decryption, the audio file is often a raw stream without a
        proper container. This method uses ffmpeg to copy the audio codec
        stream into a new, valid OGG container, making it playable.

        Raises:
            subprocess.CalledProcessError: If the ffmpeg command fails.
            Exception: For other errors during the process.
        """
        try:
            process = await asyncio.create_subprocess_exec(
                "ffmpeg",
                "-i",
                self.decrypted_file,
                "-c",
                "copy",
                self.output_file,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )
            _, stderr = await process.communicate()
            if process.returncode != 0:
                LOGGER.error("ffmpeg error: %s", stderr.decode().strip())
                raise subprocess.CalledProcessError(process.returncode, "ffmpeg")
        except Exception as e:
            LOGGER.error("Error fixing audio file: %s", e)
            raise

    async def _cleanup(self) -> None:
        """Cleans up temporary encrypted and decrypted files asynchronously."""
        for file in [self.encrypted_file, self.decrypted_file]:
            try:
                if os.path.exists(file):
                    os.remove(file)
            except Exception as e:
                LOGGER.warning("Error removing %s: %s", file, e)

    async def process(self) -> Union[Path, types.Error]:
        """The main processing pipeline for a Spotify track.

        This function orchestrates the entire process:
        1. Downloads the encrypted audio file.
        2. Decrypts the audio.
        3. Rebuilds the OGG headers of the decrypted file.
        4. Uses ffmpeg to fix the container.
        5. Cleans up temporary files.

        If the final output file already exists, it skips the process and
        returns the path immediately.

        Returns:
            Union[Path, types.Error]: The path to the final, playable audio
                file, or an `Error` object if any step in the process fails.
        """
        if os.path.exists(self.output_file):
            LOGGER.info("✅ Found existing file: %s", self.output_file)
            return Path(self.output_file)

        _track_id = self.track.tc
        if not self.track.cdnurl or not self.track.key:
            LOGGER.warning("Missing CDN URL or key for track: %s", _track_id)
            return types.Error(
                code=400, message=f"Missing CDN URL or key for track: {_track_id}"
            )

        try:
            await HttpxClient().download_file(self.track.cdnurl, self.encrypted_file)
            await self.decrypt_audio()
            await rebuild_ogg(self.decrypted_file)
            await self.fix_audio()
            await self._cleanup()
            LOGGER.info("✅ Successfully processed track: %s", self.output_file)
            return Path(self.output_file)
        except Exception as e:
            LOGGER.error("Error processing track %s: %s", _track_id, e)
            await self._cleanup()
            return types.Error(code=500, message=f"Error processing track: {_track_id}")
