/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"context"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetAssistant retrieves the index of the assistant for a chat.
func (db *Database) GetAssistant(ctx context.Context, chatID int64) (int, error) {
	key := toKey(chatID)
	if cached, ok := db.assistantCache.Get(key); ok {
		return cached, nil
	}
	var doc struct {
		Num int `bson:"num"`
	}
	err := db.assistantDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	db.assistantCache.Set(key, doc.Num)
	return doc.Num, nil
}

// SetAssistant sets the assistant index for a given chat.
func (db *Database) SetAssistant(ctx context.Context, chatID int64, num int) error {
	_, err := db.assistantDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{"num": num}}, options.UpdateOne().SetUpsert(true))
	if err == nil {
		db.assistantCache.Set(toKey(chatID), num)
	}
	return err
}

// RemoveAssistant removes the assistant from a chat's settings.
func (db *Database) RemoveAssistant(ctx context.Context, chatID int64) error {
	_, err := db.assistantDB.DeleteOne(ctx, bson.M{"_id": chatID})
	if err == nil {
		db.assistantCache.Delete(toKey(chatID))
	}
	return err
}

// AssignAssistant attempts to set the assistant for a chat if it is not currently set.
func (db *Database) AssignAssistant(ctx context.Context, chatID int64, proposedAssistant int) (int, error) {
	filter := bson.M{
		"_id": chatID,
		"$or": bson.A{
			bson.M{"num": bson.M{"$exists": false}},
			bson.M{"num": 0},
		},
	}
	update := bson.M{"$set": bson.M{"num": proposedAssistant}}
	opts := options.UpdateOne().SetUpsert(true)

	result, err := db.assistantDB.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return db.GetAssistant(ctx, chatID)
		}
		return 0, err
	}

	if result.ModifiedCount > 0 || result.UpsertedCount > 0 {
		db.assistantCache.Set(toKey(chatID), proposedAssistant)
		return proposedAssistant, nil
	}

	return db.GetAssistant(ctx, chatID)
}

// ClearAllAssistants removes all assistant assignments.
func (db *Database) ClearAllAssistants(ctx context.Context) (int64, error) {
	result, err := db.assistantDB.DeleteMany(ctx, bson.M{})
	if err != nil {
		slog.Info("[DB] Error clearing assistants", "error", err)
		return 0, err
	}
	db.assistantCache.Clear()
	return result.DeletedCount, nil
}
