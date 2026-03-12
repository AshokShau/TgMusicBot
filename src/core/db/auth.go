/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"ashokshau/tgmusic/src/core/cache"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// AddAuthUser adds a user to the list of authorized users for a chat.
func (db *Database) AddAuthUser(ctx context.Context, chatID, userID int64) error {
	_, err := db.authDB.UpdateOne(ctx,
		bson.M{"_id": chatID},
		bson.M{"$addToSet": bson.M{"user_ids": userID}},
		options.UpdateOne().SetUpsert(true),
	)
	if err == nil {
		db.authCache.Delete(toKey(chatID))
	}
	return err
}

// RemoveAuthUser removes a user from the list of authorized users for a chat.
func (db *Database) RemoveAuthUser(ctx context.Context, chatID, userID int64) error {
	_, err := db.authDB.UpdateOne(ctx,
		bson.M{"_id": chatID},
		bson.M{"$pull": bson.M{"user_ids": userID}},
	)
	if err == nil {
		db.authCache.Delete(toKey(chatID))
	}
	return err
}

// GetAuthUsers retrieves a list of all authorized users for a chat.
func (db *Database) GetAuthUsers(ctx context.Context, chatID int64) []int64 {
	key := toKey(chatID)
	if cached, ok := db.authCache.Get(key); ok {
		return cached
	}
	var doc struct {
		UserIDs []int64 `bson:"user_ids"`
	}
	err := db.authDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc)
	if err != nil {
		return []int64{}
	}
	db.authCache.Set(key, doc.UserIDs)
	return doc.UserIDs
}

// IsAuthUser checks if a specific user is in the list of authorized users for a chat.
func (db *Database) IsAuthUser(ctx context.Context, chatID, userID int64) bool {
	admins, err := cache.GetChatAdminIDs(chatID)
	if err != nil || admins == nil {
		admins = []int64{}
	}

	if contains(admins, userID) {
		return true
	}

	users := db.GetAuthUsers(ctx, chatID)
	return contains(users, userID)
}

// IsAdmin checks if a specific user is an administrator in a chat.
func (db *Database) IsAdmin(_ context.Context, chatID, userID int64) bool {
	admins, err := cache.GetChatAdminIDs(chatID)
	if err != nil || admins == nil {
		admins = []int64{}
	}
	return contains(admins, userID)
}
