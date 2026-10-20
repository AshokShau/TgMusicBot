package cache

import (
	"errors"

	td "github.com/AshokShau/gotdbot"
)

// SettingsAccess is shared by Telegram callbacks and web-app settings.
func SettingsAccess(c *td.Client, chatID, userID int64, setting string) error {
	rights, err := GetRights(c, chatID, userID, false)
	if err != nil || rights == nil {
		return errors.New("you must be a group administrator to change settings")
	}
	if setting == "main" {
		return nil
	}

	if setting == "silent" {
		if !rights.CanDeleteMessages {
			return errors.New("you need Delete Messages permission to change Silent Mode")
		}
		return nil
	}

	if !rights.CanManageChat {
		return errors.New("you must be a group administrator to change settings")
	}
	return nil
}

func BotCanDeleteCommands(c *td.Client, chatID int64) bool {
	if c == nil || c.Me == nil {
		return false
	}

	rights, err := GetRights(c, chatID, c.Me.Id, false)
	return err == nil && rights != nil && rights.CanDeleteMessages
}
