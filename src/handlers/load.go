/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

var startTime = time.Now()
var logger tg.Logger

// LoadModules loads all the handlers.
// It takes a telegram client as input.
func LoadModules(c *tg.Client) {
	_, _ = c.UpdatesGetState()
	logger = c.Log

	c.On("command:ping", pingHandler)
	c.On("command:start", startHandler)
	c.On("command:help", startHandler)
	c.On("command:lang", langHandler)
	c.On("command:reload", reloadAdminCacheHandler)
	c.On("command:privacy", privacyHandler)

	c.On("command:play", playHandler, tg.FilterFunc(playMode))
	c.On("command:vPlay", vPlayHandler, tg.FilterFunc(playMode))

	c.On("command:loop", loopHandler, tg.FilterFunc(adminMode))
	c.On("command:remove", removeHandler, tg.FilterFunc(adminMode))
	c.On("command:skip", skipHandler, tg.FilterFunc(adminMode))
	c.On("command:stop", stopHandler, tg.FilterFunc(adminMode))
	c.On("command:end", stopHandler, tg.FilterFunc(adminMode))
	c.On("command:mute", muteHandler, tg.FilterFunc(adminMode))
	c.On("command:unmute", unmuteHandler, tg.FilterFunc(adminMode))
	c.On("command:pause", pauseHandler, tg.FilterFunc(adminMode))
	c.On("command:resume", resumeHandler, tg.FilterFunc(adminMode))
	c.On("command:queue", queueHandler, tg.FilterFunc(adminMode))
	c.On("command:seek", seekHandler, tg.FilterFunc(adminMode))
	c.On("command:speed", speedHandler, tg.FilterFunc(adminMode))
	c.On("command:authList", authListHandler, tg.FilterFunc(adminMode))
	c.On("command:addAuth", addAuthHandler, tg.FilterFunc(adminMode))
	c.On("command:auth", addAuthHandler, tg.FilterFunc(adminMode))
	c.On("command:removeAuth", removeAuthHandler, tg.FilterFunc(adminMode))
	c.On("command:unAuth", removeAuthHandler, tg.FilterFunc(adminMode))
	c.On("command:rmAuth", removeAuthHandler, tg.FilterFunc(adminMode))

	c.On("command:active_vc", activeVcHandler, tg.FilterFunc(isDev))
	c.On("command:av", activeVcHandler, tg.FilterFunc(isDev))
	c.On("command:stats", sysStatsHandler, tg.FilterFunc(isDev))
	c.On("command:clear_assistants", clearAssistantsHandler, tg.FilterFunc(isDev))
	c.On("command:clearAss", clearAssistantsHandler, tg.FilterFunc(isDev))
	c.On("command:leaveAll", leaveAllHandler, tg.FilterFunc(isDev))
	c.On("command:broadcast", broadcastHandler, tg.FilterFunc(isDev))
	c.On("command:gCast", broadcastHandler, tg.FilterFunc(isDev))
	c.On("command:cancelBroadcast", cancelBroadcastHandler, tg.FilterFunc(isDev))

	c.On("command:settings", settingsHandler, tg.FilterFunc(adminMode))
	c.On("callback:play_\\w+", playCallbackHandler, tg.FilterFuncCallback(adminModeCB))
	c.On("callback:vcplay_\\w+", vcPlayHandler)
	c.On("callback:help_\\w+", helpCallbackHandler)
	c.On("callback:settings_\\w+", settingsCallbackHandler)
	c.On("callback:setlang_\\w+", setLangCallbackHandler)

	c.AddParticipantHandler(handleParticipant)
	c.AddActionHandler(handleVoiceChatMessage)
	logger.Debug("Handlers loaded successfully.")
}
