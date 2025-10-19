package db

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/AshokShau/TgMusicBot/pkg/config"
	"github.com/AshokShau/TgMusicBot/pkg/core/cache"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database encapsulates the MongoDB connection, database, collections, and caches.
type Database struct {
	Client    *mongo.Client
	DB        *mongo.Database
	ChatDB    *mongo.Collection
	UserDB    *mongo.Collection
	BotDB     *mongo.Collection
	ChatCache *cache.Cache[map[string]interface{}]
	BotCache  *cache.Cache[map[string]interface{}]
	UserCache *cache.Cache[map[string]interface{}]
}

// Instance is the global singleton for the database.
var Instance *Database

// InitDatabase initializes the database connection and sets up the global instance.
// It returns an error if the connection fails or pinging the database is unsuccessful.
func InitDatabase(ctx context.Context) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.Conf.MongoUri))
	if err != nil {
		return err
	}

	db := client.Database(config.Conf.DbName)

	Instance = &Database{
		Client:    client,
		DB:        db,
		ChatDB:    db.Collection("chats"),
		UserDB:    db.Collection("users"),
		BotDB:     db.Collection("bot"),
		ChatCache: cache.NewCache[map[string]interface{}](20 * time.Minute),
		BotCache:  cache.NewCache[map[string]interface{}](20 * time.Minute),
		UserCache: cache.NewCache[map[string]interface{}](20 * time.Minute),
	}

	if err := Instance.Ping(ctx); err != nil {
		return err
	}

	log.Println("[DB] The database connection has been successfully established.")
	return nil
}

// Ping verifies the connection to the MongoDB server.
// It returns an error if the connection is not active.
func (db *Database) Ping(ctx context.Context) error {
	return db.Client.Ping(ctx, nil)
}

// ----------------- CHAT -----------------

// GetChat retrieves a chat's data from the cache or database.
// It returns a map representing the chat data, or nil if not found.
func (db *Database) GetChat(ctx context.Context, chatID int64) (map[string]interface{}, error) {
	key := toKey(chatID)
	if cached, ok := db.ChatCache.Get(key); ok {
		return cached, nil
	}

	var chat map[string]interface{}
	err := db.ChatDB.FindOne(ctx, bson.M{"_id": chatID}).Decode(&chat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	} else if err != nil {
		log.Printf("[DB] An error occurred while getting the chat: %v", err)
		return nil, err
	}

	db.ChatCache.Set(key, chat)
	return chat, nil
}

// AddChat adds a new chat to the database if it does not already exist.
func (db *Database) AddChat(ctx context.Context, chatID int64) error {
	chat, _ := db.GetChat(ctx, chatID)
	if chat != nil {
		return nil // Chat already exists.
	}
	_, err := db.ChatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$setOnInsert": bson.M{}}, options.Update().SetUpsert(true))
	if err == nil {
		log.Printf("[DB] A new chat has been added: %d", chatID)
	}
	return err
}

// updateChatField updates a specific field in a chat's document.
func (db *Database) updateChatField(ctx context.Context, chatID int64, key string, value interface{}) error {
	_, err := db.ChatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{key: value}}, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	cached, _ := db.ChatCache.Get(toKey(chatID))
	if cached == nil {
		cached = make(map[string]interface{})
	}
	cached[key] = value
	db.ChatCache.Set(toKey(chatID), cached)
	return nil
}

// GetPlayType retrieves the play type setting for a chat.
// It returns 0 if no play type is set.
func (db *Database) GetPlayType(ctx context.Context, chatID int64) int {
	chat, _ := db.GetChat(ctx, chatID)
	if chat == nil {
		return 0
	}
	if val, ok := chat["play_type"].(int32); ok {
		return int(val)
	}
	return 0
}

// SetPlayType sets the play type for a given chat.
func (db *Database) SetPlayType(ctx context.Context, chatID int64, playType int) error {
	return db.updateChatField(ctx, chatID, "play_type", playType)
}

// GetPlayMode retrieves the play mode for a chat.
// It returns "everyone" by default.
func (db *Database) GetPlayMode(ctx context.Context, chatID int64) string {
	chat, _ := db.GetChat(ctx, chatID)
	if chat == nil {
		return "everyone"
	}
	if val, ok := chat["play_mode"].(string); ok {
		return val
	}
	return "everyone"
}

// SetPlayMode sets the play mode for a given chat.
func (db *Database) SetPlayMode(ctx context.Context, chatID int64, playMode string) error {
	return db.updateChatField(ctx, chatID, "play_mode", playMode)
}

// GetAdminMode retrieves the admin mode for a chat.
// It returns "everyone" by default.
func (db *Database) GetAdminMode(ctx context.Context, chatID int64) string {
	chat, _ := db.GetChat(ctx, chatID)
	if chat == nil {
		return "everyone"
	}
	if val, ok := chat["admin_mode"].(string); ok {
		return val
	}
	return "everyone"
}

// SetAdminMode sets the admin mode for a given chat.
func (db *Database) SetAdminMode(ctx context.Context, chatID int64, adminMode string) error {
	return db.updateChatField(ctx, chatID, "admin_mode", adminMode)
}

// GetAssistant retrieves the username of the assistant for a chat.
func (db *Database) GetAssistant(ctx context.Context, chatID int64) (string, error) {
	chat, _ := db.GetChat(ctx, chatID)
	if chat == nil {
		return "", nil
	}
	if val, ok := chat["assistant"].(string); ok {
		return val, nil
	}
	return "", nil
}

// SetAssistant sets the assistant for a given chat.
func (db *Database) SetAssistant(ctx context.Context, chatID int64, assistant string) error {
	return db.updateChatField(ctx, chatID, "assistant", assistant)
}

// RemoveAssistant removes the assistant from a chat's settings.
func (db *Database) RemoveAssistant(ctx context.Context, chatID int64) error {
	return db.updateChatField(ctx, chatID, "assistant", nil)
}

// ----------------- AUTH USERS -----------------

// AddAuthUser adds a user to the list of authorized users for a chat.
func (db *Database) AddAuthUser(ctx context.Context, chatID, userID int64) error {
	_, err := db.ChatDB.UpdateOne(ctx,
		bson.M{"_id": chatID},
		bson.M{"$addToSet": bson.M{"auth_users": userID}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}
	chat, _ := db.GetChat(ctx, chatID)
	authUsers, _ := getIntSlice(chat["auth_users"])
	if !contains(authUsers, userID) {
		authUsers = append(authUsers, userID)
	}
	chat["auth_users"] = authUsers
	db.ChatCache.Set(toKey(chatID), chat)
	return nil
}

// RemoveAuthUser removes a user from the list of authorized users for a chat.
func (db *Database) RemoveAuthUser(ctx context.Context, chatID, userID int64) error {
	_, err := db.ChatDB.UpdateOne(ctx,
		bson.M{"_id": chatID},
		bson.M{"$pull": bson.M{"auth_users": userID}},
	)
	if err != nil {
		return err
	}
	chat, _ := db.GetChat(ctx, chatID)
	authUsers, _ := getIntSlice(chat["auth_users"])
	authUsers = remove(authUsers, userID)
	chat["auth_users"] = authUsers
	db.ChatCache.Set(toKey(chatID), chat)
	return nil
}

// GetAuthUsers retrieves a list of all authorized users for a chat.
func (db *Database) GetAuthUsers(ctx context.Context, chatID int64) []int64 {
	chat, _ := db.GetChat(ctx, chatID)
	users, _ := getIntSlice(chat["auth_users"])
	return users
}

// IsAuthUser checks if a specific user is in the list of authorized users for a chat.
func (db *Database) IsAuthUser(ctx context.Context, chatID, userID int64) bool {
	admins, err := cache.GetChatAdmins(chatID)
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
func (db *Database) IsAdmin(ctx context.Context, chatID, userID int64) bool {
	admins, err := cache.GetChatAdmins(chatID)
	if err != nil || admins == nil {
		admins = []int64{}
	}
	return contains(admins, userID)
}

// ----------------- BOT -----------------

// GetLoggerStatus retrieves the logger status for a given bot.
// It returns true if the logger is enabled, and false otherwise.
func (db *Database) GetLoggerStatus(ctx context.Context, botID int64) bool {
	key := toKey(botID)
	if cached, ok := db.BotCache.Get(key); ok {
		if v, ok := cached["logger"].(bool); ok {
			return v
		}
	}

	var data map[string]interface{}
	_ = db.BotDB.FindOne(ctx, bson.M{"_id": botID}).Decode(&data)

	status := false
	if val, ok := data["logger"].(bool); ok {
		status = val
	}

	cached := map[string]interface{}{"logger": status}
	db.BotCache.Set(key, cached)
	return status
}

// SetLoggerStatus enables or disables the logger for a bot.
func (db *Database) SetLoggerStatus(ctx context.Context, botID int64, status bool) error {
	_, err := db.BotDB.UpdateOne(ctx,
		bson.M{"_id": botID},
		bson.M{"$set": bson.M{"logger": status}},
		options.Update().SetUpsert(true),
	)
	if err == nil {
		cached, _ := db.BotCache.Get(toKey(botID))
		if cached == nil {
			cached = map[string]interface{}{}
		}
		cached["logger"] = status
		db.BotCache.Set(toKey(botID), cached)
	}
	return err
}

// ----------------- USERS -----------------

// AddUser adds a new user to the database if they do not already exist.
func (db *Database) AddUser(ctx context.Context, userID int64) error {
	key := toKey(userID)

	// Check cache first to avoid unnecessary database operations.
	if _, ok := db.UserCache.Get(key); ok {
		return nil
	}

	// Upsert in the database to ensure the user is added.
	_, err := db.UserDB.UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$setOnInsert": bson.M{}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	// Update the cache to reflect the new user.
	db.UserCache.Set(key, map[string]interface{}{})
	return nil
}

// RemoveUser removes a user from the database and cache.
func (db *Database) RemoveUser(ctx context.Context, userID int64) error {
	key := toKey(userID)

	// Delete from the database.
	_, err := db.UserDB.DeleteOne(ctx, bson.M{"_id": userID})
	if err != nil {
		return err
	}

	// Delete from the cache.
	db.UserCache.Delete(key)
	return nil
}

// IsUserExist checks if a user exists in the database.
// It returns true if the user is found, false otherwise, and an error if one occurs.
func (db *Database) IsUserExist(ctx context.Context, userID int64) (bool, error) {
	key := toKey(userID)

	// Check the cache first.
	if _, ok := db.UserCache.Get(key); ok {
		return true, nil
	}

	// If not in cache, check the database.
	var result bson.M
	err := db.UserDB.FindOne(ctx, bson.M{"_id": userID}).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	// If found, add to cache.
	db.UserCache.Set(key, map[string]interface{}{})
	return true, nil
}

// GetAllChats retrieves a list of all chat IDs from the database.
func (db *Database) GetAllChats(ctx context.Context) ([]int64, error) {
	cursor, err := db.ChatDB.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chats []int64
	for cursor.Next(ctx) {
		var doc struct {
			ID int64 `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		chats = append(chats, doc.ID)

		// Cache each chat to optimize future lookups.
		db.ChatCache.Set(toKey(doc.ID), map[string]interface{}{})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return chats, nil
}

// GetAllUsers retrieves a list of all user IDs from the database.
func (db *Database) GetAllUsers(ctx context.Context) ([]int64, error) {
	cursor, err := db.UserDB.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []int64
	for cursor.Next(ctx) {
		var doc struct {
			ID int64 `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		users = append(users, doc.ID)

		// Cache each user to optimize future lookups.
		db.UserCache.Set(toKey(doc.ID), map[string]interface{}{})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// Close gracefully closes the database connection.
func (db *Database) Close(ctx context.Context) error {
	log.Println("[DB] Closing the database connection...")
	return db.Client.Disconnect(ctx)
}
