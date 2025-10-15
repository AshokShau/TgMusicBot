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
    def __init__(self):
        self.mongo_client = AsyncMongoClient(config.MONGO_URI)
        _db = self.mongo_client[config.DB_NAME]
        self.chat_db = _db["chats"]
        self.users_db = _db["users"]
        self.bot_db = _db["bot"]

        self.chat_cache = TTLCache(maxsize=1000, ttl=1200)
        self.bot_cache = TTLCache(maxsize=1000, ttl=1200)

    async def ping(self) -> None:
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
        if await self.get_chat(chat_id) is None:
            LOGGER.info("Added chat: %s", chat_id)
            await self.chat_db.update_one(
                {"_id": chat_id}, {"$setOnInsert": {}}, upsert=True
            )

    async def _update_chat_field(self, chat_id: int, key: str, value) -> None:
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$set": {key: value}}, upsert=True
        )
        cached = self.chat_cache.get(chat_id, {})
        cached[key] = value
        self.chat_cache[chat_id] = cached

    async def get_play_type(self, chat_id: int) -> int:
        chat = await self.get_chat(chat_id)
        return chat.get("play_type", 0) if chat else 0

    async def set_play_type(self, chat_id: int, play_type: int) -> None:
        await self._update_chat_field(chat_id, "play_type", play_type)

    async def get_assistant(self, chat_id: int) -> Optional[str]:
        chat = await self.get_chat(chat_id)
        return chat.get("assistant") if chat else None

    async def set_assistant(self, chat_id: int, assistant: str) -> None:
        await self._update_chat_field(chat_id, "assistant", assistant)

    async def clear_all_assistants(self) -> int:
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
        await self._update_chat_field(chat_id, "assistant", None)

    async def add_auth_user(self, chat_id: int, auth_user: int) -> None:
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$addToSet": {"auth_users": auth_user}}, upsert=True
        )
        chat = await self.get_chat(chat_id)
        auth_users = chat.get("auth_users", [])
        if auth_user not in auth_users:
            auth_users.append(auth_user)
        self.chat_cache[chat_id]["auth_users"] = auth_users

    async def remove_auth_user(self, chat_id: int, auth_user: int) -> None:
        await self.chat_db.update_one(
            {"_id": chat_id}, {"$pull": {"auth_users": auth_user}}
        )
        chat = await self.get_chat(chat_id)
        auth_users = chat.get("auth_users", [])
        if auth_user in auth_users:
            auth_users.remove(auth_user)
        self.chat_cache[chat_id]["auth_users"] = auth_users

    async def reset_auth_users(self, chat_id: int) -> None:
        await self._update_chat_field(chat_id, "auth_users", [])

    async def get_auth_users(self, chat_id: int) -> list[int]:
        chat = await self.get_chat(chat_id)
        return chat.get("auth_users", []) if chat else []

    async def is_auth_user(self, chat_id: int, user_id: int) -> bool:
        return user_id in await self.get_auth_users(chat_id)

    async def set_buttons_status(self, chat_id: int, status: bool) -> None:
        await self._update_chat_field(chat_id, "buttons", status)

    async def get_buttons_status(self, chat_id: int) -> bool:
        chat = await self.get_chat(chat_id)
        return chat.get("buttons", True) if chat else True

    async def set_thumbnail_status(self, chat_id: int, status: bool) -> None:
        await self._update_chat_field(chat_id, "thumb", status)

    async def get_thumbnail_status(self, chat_id: int) -> bool:
        chat = await self.get_chat(chat_id)
        return chat.get("thumb", True) if chat else True

    async def remove_chat(self, chat_id: int) -> None:
        await self.chat_db.delete_one({"_id": chat_id})
        self.chat_cache.pop(chat_id, None)

    async def add_user(self, user_id: int) -> None:
        await self.users_db.update_one(
            {"_id": user_id}, {"$setOnInsert": {}}, upsert=True
        )

    async def remove_user(self, user_id: int) -> None:
        await self.users_db.delete_one({"_id": user_id})

    async def is_user_exist(self, user_id: int) -> bool:
        return await self.users_db.find_one({"_id": user_id}) is not None

    async def get_all_users(self) -> list[int]:
        return [user["_id"] async for user in self.users_db.find()]

    async def get_all_chats(self) -> list[int]:
        return [chat["_id"] async for chat in self.chat_db.find()]

    async def get_logger_status(self, bot_id: int) -> bool:
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
        await self.bot_db.update_one(
            {"_id": bot_id}, {"$set": {"logger": status}}, upsert=True
        )

        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["logger"] = status
        self.bot_cache[bot_id] = cached

    async def get_auto_end(self, bot_id: int) -> bool:
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
        await self.bot_db.update_one(
            {"_id": bot_id}, {"$set": {"auto_end": status}}, upsert=True
        )
        # Update cache
        cached = self.bot_cache.get(bot_id, {})
        cached["auto_end"] = status
        self.bot_cache[bot_id] = cached

    async def close(self) -> None:
        await self.mongo_client.close()
        LOGGER.info("Database connection closed.")


db: Database = Database()
