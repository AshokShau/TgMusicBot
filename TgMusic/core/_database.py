#  Copyright (c) 2025 AshokShau
#  Licensed under the GNU AGPL v3.0: https://www.gnu.org/licenses/agpl-3.0.html
#  Part of the TgMusicBot project. All rights reserved where applicable.

from typing import Optional

from cachetools import TTLCache
from pymongo import AsyncMongoClient
from pymongo.errors import ConnectionFailure

from TgMusic.logger import LOGGER

from ._config import config


class Database:
    """Handles all database interactions for the bot.

    This class provides an abstraction layer for the MongoDB database,
    managing collections for chats, users, and bot settings. It includes
    caching mechanisms to reduce database load for frequently accessed data.

    Attributes:
        mongo_client: An instance of `AsyncMongoClient` for database connection.
        chat_db: The collection for storing chat-specific data.
        users_db: The collection for storing user data.
        bot_db: The collection for storing global bot settings.
        chat_cache: A TTL cache for chat data.
        bot_cache: A TTL cache for bot settings.
    """

    def __init__(self):
        """Initializes the database connection and collections."""
        self.mongo_client = AsyncMongoClient(config.MONGO_URI)
        _db = self.mongo_client[config.DB_NAME]
        self.chat_db = _db["chats"]
        self.users_db = _db["users"]
        self.bot_db = _db["bot"]

        self.chat_cache = TTLCache(maxsize=1000, ttl=1200)
        self.bot_cache = TTLCache(maxsize=1000, ttl=1200)

    async def ping(self) -> None:
        """Checks the database connection by sending a ping command.

        Raises:
            ConnectionFailure: If the database server is not available.
            RuntimeError: For other database connection errors.
        """
        try:
            await self.mongo_client.aconnect()
            await self.mongo_client.admin.command("ping")
            LOGGER.info("Database connection completed.")
        except ConnectionFailure as e:
            raise ConnectionFailure(
                "Database connection failed : Server not available"
            ) from e
        except Exception as e:
            LOGGER.error("Database connection failed: %s", e)
            raise RuntimeError(f"Database connection failed.{str(e)}") from e

    async def get_chat(self, chat_id: int) -> Optional[dict]:
        """Retrieves chat data from the cache or database.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            Optional[dict]: The chat's data document, or None if not found.
        """
        if chat_id in self.chat_cache:
            return self.chat_cache[chat_id]
        try:
            if chat := await self.chat_db.find_one({"_id": chat_id}):
                self.chat_cache[chat_id] = chat
            return chat
        except Exception as e:
            LOGGER.warning("Error getting chat: %s", e)
            return None

    async def add_chat(self, chat_id: int) -> None:
        """Adds a new chat to the database if it doesn't already exist.

        Args:
            chat_id (int): The ID of the chat to add.
        """
        if await self.get_chat(chat_id) is None:
            LOGGER.info("Added chat: %s", chat_id)
            await self.chat_db.update_one(
                {"_id": chat_id}, {"$setOnInsert": {}}, upsert=True
            )

    async def _update_chat_field(self, chat_id: int, key: str, value) -> None:
        """A helper method to update a specific field in a chat's document.

        This method also updates the corresponding value in the cache.

        Args:
            chat_id (int): The ID of the chat to update.
            key (str): The field key to update.
            value: The new value for the field.
        """
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$set": {key: value}}, upsert=True
        )
        cached = self.chat_cache.get(chat_id, {})
        cached[key] = value
        self.chat_cache[chat_id] = cached

    async def get_play_type(self, chat_id: int) -> int:
        """Gets the play type (e.g., group or channel) for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            int: The play type (0 for group, 1 for channel, defaults to 0).
        """
        chat = await self.get_chat(chat_id)
        return chat.get("play_type", 0) if chat else 0

    async def set_play_type(self, chat_id: int, play_type: int) -> None:
        """Sets the play type for a chat.

        Args:
            chat_id (int): The ID of the chat.
            play_type (int): The play type to set.
        """
        await self._update_chat_field(chat_id, "play_type", play_type)

    async def get_assistant(self, chat_id: int) -> Optional[str]:
        """Gets the assigned assistant for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            Optional[str]: The assistant's identifier, or None if not set.
        """
        chat = await self.get_chat(chat_id)
        return chat.get("assistant") if chat else None

    async def set_assistant(self, chat_id: int, assistant: str) -> None:
        """Assigns an assistant to a chat.

        Args:
            chat_id (int): The ID of the chat.
            assistant (str): The identifier of the assistant to assign.
        """
        await self._update_chat_field(chat_id, "assistant", assistant)

    async def clear_all_assistants(self) -> int:
        """Removes the assistant assignment from all chats.

        This operation affects both the database and the cache.

        Returns:
            int: The number of chats that were updated.
        """
        # Clear assistants from all chats in the database
        result = await self.chat_db.update_many(
            {"assistant": {"$exists": True}}, {"$unset": {"assistant": ""}}
        )

        # Clear assistants from all cached chats
        for chat_id in list(self.chat_cache.keys()):
            if "assistant" in self.chat_cache[chat_id]:
                self.chat_cache[chat_id]["assistant"] = None

        LOGGER.info(f"Cleared assistants from {result.modified_count} chats")
        return result.modified_count

    async def remove_assistant(self, chat_id: int) -> None:
        """Removes the assistant assignment from a specific chat.

        Args:
            chat_id (int): The ID of the chat.
        """
        await self._update_chat_field(chat_id, "assistant", None)

    async def add_auth_user(self, chat_id: int, auth_user: int) -> None:
        """Adds a user to the list of authorized users for a chat.

        Args:
            chat_id (int): The ID of the chat.
            auth_user (int): The ID of the user to authorize.
        """
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$addToSet": {"auth_users": auth_user}}, upsert=True
        )
        chat = await self.get_chat(chat_id)
        auth_users = chat.get("auth_users", [])
        if auth_user not in auth_users:
            auth_users.append(auth_user)
        self.chat_cache[chat_id]["auth_users"] = auth_users

    async def remove_auth_user(self, chat_id: int, auth_user: int) -> None:
        """Removes a user from the list of authorized users for a chat.

        Args:
            chat_id (int): The ID of the chat.
            auth_user (int): The ID of the user to de-authorize.
        """
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$pull": {"auth_users": auth_user}}
        )
        chat = await self.get_chat(chat_id)
        auth_users = chat.get("auth_users", [])
        if auth_user in auth_users:
            auth_users.remove(auth_user)
        self.chat_cache[chat_id]["auth_users"] = auth_users

    async def reset_auth_users(self, chat_id: int) -> None:
        """Removes all authorized users from a chat.

        Args:
            chat_id (int): The ID of the chat.
        """
        await self._update_chat_field(chat_id, "auth_users", [])

    async def get_auth_users(self, chat_id: int) -> list[int]:
        """Gets the list of authorized user IDs for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            list[int]: A list of authorized user IDs.
        """
        chat = await self.get_chat(chat_id)
        return chat.get("auth_users", []) if chat else []

    async def is_auth_user(self, chat_id: int, user_id: int) -> bool:
        """Checks if a user is authorized in a chat.

        Args:
            chat_id (int): The ID of the chat.
            user_id (int): The ID of the user.

        Returns:
            bool: True if the user is authorized, False otherwise.
        """
        return user_id in await self.get_auth_users(chat_id)

    async def set_buttons_status(self, chat_id: int, status: bool) -> None:
        """Enables or disables control buttons for a chat.

        Args:
            chat_id (int): The ID of the chat.
            status (bool): The new status for the buttons.
        """
        await self._update_chat_field(chat_id, "buttons", status)

    async def get_buttons_status(self, chat_id: int) -> bool:
        """Checks if control buttons are enabled for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            bool: True if buttons are enabled, False otherwise. Defaults to True.
        """
        chat = await self.get_chat(chat_id)
        return chat.get("buttons", True) if chat else True

    async def set_thumbnail_status(self, chat_id: int, status: bool) -> None:
        """Enables or disables thumbnails for a chat.

        Args:
            chat_id (int): The ID of the chat.
            status (bool): The new status for thumbnails.
        """
        await self._update_chat_field(chat_id, "thumb", status)

    async def get_thumbnail_status(self, chat_id: int) -> bool:
        """Checks if thumbnails are enabled for a chat.

        Args:
            chat_id (int): The ID of the chat.

        Returns:
            bool: True if thumbnails are enabled, False otherwise. Defaults to True.
        """
        chat = await self.get_chat(chat_id)
        return chat.get("thumb", True) if chat else True

    async def remove_chat(self, chat_id: int) -> None:
        """Deletes a chat from the database and cache.

        Args:
            chat_id (int): The ID of the chat to remove.
        """
        await self.chat_db.delete_one({"_id": chat_id})
        self.chat_cache.pop(chat_id, None)

    async def add_user(self, user_id: int) -> None:
        """Adds a new user to the database if they don't already exist.

        Args:
            user_id (int): The ID of the user to add.
        """
        await self.users_db.update_one(
            {"_id": user_id}, {"$setOnInsert": {}}, upsert=True
        )

    async def remove_user(self, user_id: int) -> None:
        """Removes a user from the database.

        Args:
            user_id (int): The ID of the user to remove.
        """
        await self.users_db.delete_one({"_id": user_id})

    async def is_user_exist(self, user_id: int) -> bool:
        """Checks if a user exists in the database.

        Args:
            user_id (int): The ID of the user.

        Returns:
            bool: True if the user exists, False otherwise.
        """
        return await self.users_db.find_one({"_id": user_id}) is not None

    async def get_all_users(self) -> list[int]:
        """Retrieves a list of all user IDs from the database.

        Returns:
            list[int]: A list of all user IDs.
        """
        return [user["_id"] async for user in self.users_db.find()]

    async def get_all_chats(self) -> list[int]:
        """Retrieves a list of all chat IDs from the database.

        Returns:
            list[int]: A list of all chat IDs.
        """
        return [chat["_id"] async for chat in self.chat_db.find()]

    async def get_logger_status(self, bot_id: int) -> bool:
        """Gets the logger status for a specific bot instance.

        Args:
            bot_id (int): The ID of the bot.

        Returns:
            bool: The logger status (enabled/disabled). Defaults to False.
        """
        if bot_id in self.bot_cache and self.bot_cache[bot_id].get("logger"):
            return self.bot_cache[bot_id].get("logger")

        bot_data = await self.bot_db.find_one({"_id": bot_id})
        status = bot_data.get("logger", False) if bot_data else False

        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["logger"] = status
        self.bot_cache[bot_id] = cached

        return status

    async def set_logger_status(self, bot_id: int, status: bool) -> None:
        """Sets the logger status for a specific bot instance.

        Args:
            bot_id (int): The ID of the bot.
            status (bool): The new logger status.
        """
        await self.bot_db.update_one(
            {"_id": bot_id}, {"$set": {"logger": status}}, upsert=True
        )

        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["logger"] = status
        self.bot_cache[bot_id] = cached

    async def get_auto_end(self, bot_id: int) -> bool:
        """Checks if the auto-end stream feature is enabled for a bot.

        Args:
            bot_id (int): The ID of the bot.

        Returns:
            bool: The auto-end status. Defaults to True.
        """
        if bot_id in self.bot_cache and self.bot_cache[bot_id].get("auto_end"):
            return self.bot_cache[bot_id].get("auto_end")

        bot_data = await self.bot_db.find_one({"_id": bot_id})
        status = bot_data.get("auto_end", True) if bot_data else True
        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["auto_end"] = status
        self.bot_cache[bot_id] = cached
        return status

    async def set_auto_end(self, bot_id: int, status: bool) -> None:
        """Sets the auto-end stream feature for a bot.

        Args:
            bot_id (int): The ID of the bot.
            status (bool): The new auto-end status.
        """
        await self.bot_db.update_one(
            {"_id": bot_id}, {"$set": {"auto_end": status}}, upsert=True
        )
        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["auto_end"] = status
        self.bot_cache[bot_id] = cached

    async def close(self) -> None:
        """Closes the database connection."""
        await self.mongo_client.close()
        LOGGER.info("Database connection closed.")


db: Database = Database()
