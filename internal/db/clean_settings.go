package db

import (
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const DefaultAutoDeleteMinutes = 2

func ValidateAutoDeleteMinutes(minutes int) error {
	if minutes < 2 || minutes > 7 {
		return errors.New("auto-delete timer must be between 2 and 7 minutes")
	}
	return nil
}

// CleanSettings is independent of command deletion and playback authorization.
type CleanSettings struct {
	AutoDelete bool
	Minutes    int
	Silent     bool
}

func (db *Database) GetCleanSettings(chatID int64) (CleanSettings, error) {
	settings := CleanSettings{Minutes: DefaultAutoDeleteMinutes}
	if chatID >= 0 {
		return settings, nil
	}
	chat, err := db.getChat(chatID)
	if err != nil || chat == nil {
		return settings, err
	}
	settings.AutoDelete, settings.Silent = chat.AutoDelete, chat.SilentMode
	if ValidateAutoDeleteMinutes(chat.AutoDeleteMinutes) == nil {
		settings.Minutes = chat.AutoDeleteMinutes
	}
	return settings, nil
}

// Each setter updates only its field, so concurrent independent changes don't overwrite one another.
func (db *Database) setCleanField(chatID int64, field string, value any) error {
	if chatID >= 0 {
		return errors.New("clean settings are available only in groups")
	}
	ctx, cancel := db.ctx()
	defer cancel()
	_, err := db.chatDB.UpdateOne(ctx, bson.M{"_id": chatID}, bson.M{"$set": bson.M{field: value}}, options.UpdateOne().SetUpsert(true))
	if err == nil {
		db.chatCache.Delete(toKey(chatID))
	}
	return err
}
func (db *Database) SetAutoDelete(chatID int64, enabled bool) error {
	return db.setCleanField(chatID, "auto_delete", enabled)
}
func (db *Database) SetAutoDeleteMinutes(chatID int64, minutes int) error {
	if err := ValidateAutoDeleteMinutes(minutes); err != nil {
		return err
	}
	return db.setCleanField(chatID, "auto_delete_minutes", minutes)
}
func (db *Database) SetSilentMode(chatID int64, enabled bool) error {
	return db.setCleanField(chatID, "silent_mode", enabled)
}
