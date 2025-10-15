from pytdbot import Client, types

from ..logger import LOGGER
from ..modules.utils import sec_to_min
from ._config import config
from ._dataclass import CachedTrack


async def send_logger(client: Client, chat_id, song: CachedTrack):
    """
    Send a message to the logger channel when a song is played.

    Args:
        client (Client): The client to send the message with.
        chat_id (int): The ID of the chat that the song is being played in.
        song (CachedTrack): The song that is being played.

    Returns:
        None
    """
    if not chat_id or not song or chat_id == config.LOGGER_ID or config.LOGGER_ID == 0:
        return

    text = (
        f"<b>Song Playing</b> in <code>{chat_id}</code>\n\n"
        f"▶️ <b>Now Playing:</b> <a href='{song.url}'>{song.name}</a>\n\n"
        f"• <b>Duration:</b> {sec_to_min(song.duration)}\n"
        f"• <b>Requested by:</b> {song.user}\n"
        f"• <b>Platform:</b> {song.platform}"
    )

    msg = await client.sendTextMessage(
        config.LOGGER_ID, text, disable_web_page_preview=True, disable_notification=True
    )
    if isinstance(msg, types.Error):
        LOGGER.error("Error sending message: %s", msg)
    return
