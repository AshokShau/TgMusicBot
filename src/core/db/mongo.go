/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"ashokshau/tgmusic/config"
	"context"
	"errors"
	"log/slog"
	"time"

	"ashokshau/tgmusic/src/core/cache"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Database encapsulates the MongoDB connection, database, collections, and caches.
type Database struct {
	client       *mongo.Client
	DB           *mongo.Database
	chatDB       *mongo.Collection
	userDB       *mongo.Collection
	playlistDB   *mongo.Collection
	assistantDB  *mongo.Collection
	authDB       *mongo.Collection
	langDB       *mongo.Collection
	cacheDB      *mongo.Collection

	chatCache      *cache.Cache[*Chats]
	userCache      *cache.Cache[*Users]
	assistantCache *cache.Cache[int]
	authCache      *cache.Cache[[]int64]
	langCache      *cache.Cache[string]
	loggerCache    *cache.Cache[bool]
	blChatsCache   *cache.Cache[[]int64]
	blUsersCache   *cache.Cache[[]int64]
}

// Instance is the global singleton for the database.
var Instance *Database

// InitDatabase initializes the database connection and sets up the global instance.
func InitDatabase(ctx context.Context) error {
	opts := options.Client().ApplyURI(config.Conf.MongoUri).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(10 * time.Minute).
		SetConnectTimeout(20 * time.Second)

	client, err := mongo.Connect(opts)
	if err != nil {
		return err
	}

	db := client.Database(config.Conf.DbName)
	Instance = &Database{
		client:      client,
		DB:          db,
		chatDB:      db.Collection("chats"),
		userDB:      db.Collection("users"),
		playlistDB:  db.Collection("playlists"),
		assistantDB: db.Collection("assistant"),
		authDB:      db.Collection("auth"),
		langDB:      db.Collection("lang"),
		cacheDB:     db.Collection("cache"),

		chatCache:      cache.NewCache[*Chats](20 * time.Minute),
		userCache:      cache.NewCache[*Users](20 * time.Minute),
		assistantCache: cache.NewCache[int](20 * time.Minute),
		authCache:      cache.NewCache[[]int64](20 * time.Minute),
		langCache:      cache.NewCache[string](20 * time.Minute),
		loggerCache:    cache.NewCache[bool](20 * time.Minute),
		blChatsCache:   cache.NewCache[[]int64](20 * time.Minute),
		blUsersCache:   cache.NewCache[[]int64](20 * time.Minute),
	}

	if err := client.Ping(ctx, nil); err != nil {
		return errors.New("failed to ping database: " + err.Error())
	}

	slog.Info("[DB] The database connection has been successfully established.")
	return nil
}

// Close gracefully closes the database connection.
func (db *Database) Close(ctx context.Context) error {
	slog.Info("[DB] Closing the database connection...")
	return db.client.Disconnect(ctx)
}
