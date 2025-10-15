#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

import asyncio
import signal

from TgMusic import client


def handle_shutdown():
    """Initiates the graceful shutdown process.

    This function is designed to be used as a signal handler. It schedules
    the main `shutdown` coroutine to be run on the event loop.
    """
    client.logger.info("Shutting down...")
    asyncio.ensure_future(shutdown())


async def shutdown():
    """Performs a graceful shutdown of the bot and its components.

    This coroutine stops the main bot client, cancels all other running
    asyncio tasks, and then stops the event loop itself.
    """
    if client.is_running:
        await client.stop_task()
    tasks = [t for t in asyncio.all_tasks() if t is not asyncio.current_task()]
    for task in tasks:
        task.cancel()
    await asyncio.gather(*tasks, return_exceptions=True)
    client.loop.stop()


def main() -> None:
    """The main entry point for starting the bot.

    This function sets up signal handlers for graceful termination (SIGINT, SIGTERM),
    then starts the bot's main event loop and initialization process. It includes
    exception handling to log fatal errors and ensures a clean shutdown.
    """
    client.logger.info("Starting TgMusicBot...")

    # Set up signal handlers
    try:
        for sig in (signal.SIGINT, signal.SIGTERM):
            client.loop.add_signal_handler(sig, handle_shutdown)
    except (NotImplementedError, RuntimeError) as e:
        client.logger.warning(f"Could not set up signal handler: {e}")
        for sig in (signal.SIGINT, signal.SIGTERM):
            signal.signal(sig, lambda s, f: handle_shutdown())

    try:
        client.loop.run_until_complete(client.initialize_components())
        client.run()
    except Exception as e:
        client.logger.critical(f"Fatal error: {e}", exc_info=True)
    finally:
        if not client.loop.is_closed():
            client.loop.run_until_complete(shutdown())
        client.loop.close()


if __name__ == "__main__":
    main()
