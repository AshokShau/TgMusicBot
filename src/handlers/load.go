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
	c.On("command:setRtmp ", setRtmpHandler)

	c.On("command:play", playHandler, tg.Custom(playMode))
	c.On("command:vPlay", vPlayHandler, tg.Custom(playMode))
	c.On("command:stream", streamHandler, tg.Custom(playMode))

	c.On("command:stopStream", stopStreamHandler, tg.Custom(adminMode))
	c.On("command:loop", loopHandler, tg.Custom(adminMode))
	c.On("command:remove", removeHandler, tg.Custom(adminMode))
	c.On("command:skip", skipHandler, tg.Custom(adminMode))
	c.On("command:stop", stopHandler, tg.Custom(adminMode))
	c.On("command:end", stopHandler, tg.Custom(adminMode))
	c.On("command:mute", muteHandler, tg.Custom(adminMode))
	c.On("command:unmute", unmuteHandler, tg.Custom(adminMode))
	c.On("command:pause", pauseHandler, tg.Custom(adminMode))
	c.On("command:resume", resumeHandler, tg.Custom(adminMode))
	c.On("command:queue", queueHandler, tg.Custom(adminMode))
	c.On("command:seek", seekHandler, tg.Custom(adminMode))
	c.On("command:speed", speedHandler, tg.Custom(adminMode))
	c.On("command:authList", authListHandler, tg.Custom(adminMode))
	c.On("command:addAuth", addAuthHandler, tg.Custom(adminMode))
	c.On("command:auth", addAuthHandler, tg.Custom(adminMode))
	c.On("command:removeAuth", removeAuthHandler, tg.Custom(adminMode))
	c.On("command:unAuth", removeAuthHandler, tg.Custom(adminMode))
	c.On("command:rmAuth", removeAuthHandler, tg.Custom(adminMode))

	c.On("command:active_vc", activeVcHandler, tg.Custom(isDev))
	c.On("command:av", activeVcHandler, tg.Custom(isDev))
	c.On("command:stats", sysStatsHandler, tg.Custom(isDev))
	c.On("command:streams", listStreamsHandler, tg.Custom(isDev))
	c.On("command:clear_assistants", clearAssistantsHandler, tg.Custom(isDev))
	c.On("command:clearAss", clearAssistantsHandler, tg.Custom(isDev))
	c.On("command:leaveAll", leaveAllHandler, tg.Custom(isDev))
	c.On("command:logger", loggerHandler, tg.Custom(isDev))
	c.On("command:broadcast", broadcastHandler, tg.Custom(isDev))
	c.On("command:gCast", broadcastHandler, tg.Custom(isDev))
	c.On("command:cancelBroadcast", cancelBroadcastHandler, tg.Custom(isDev))
	c.On("command:cleanupchats", cleanupChatsHandler, tg.Custom(isDev))
	c.On("command:cleanup", cleanupChatsHandler, tg.Custom(isDev))

	c.On("command:settings", settingsHandler, tg.Custom(adminMode))

	// Force Subscribe commands (owner only, private chat)
	c.On("command:addfsub", addFsubHandler, tg.Custom(isDev))
	c.On("command:rmfsub", removeFsubHandler, tg.Custom(isDev))
	c.On("command:delfsub", removeFsubHandler, tg.Custom(isDev))
	c.On("command:fsub", fsubStatusHandler, tg.Custom(isDev))

	c.On("command:cplist", createPlaylistHandler)
	c.On("command:createplaylist", createPlaylistHandler)
	c.On("command:dlplist", deletePlaylistHandler)
	c.On("command:deleteplaylist", deletePlaylistHandler)
	c.On("command:addtoplist", addToPlaylistHandler)
	c.On("command:addtoplaylist", addToPlaylistHandler)
	c.On("command:rmplist", removeFromPlaylistHandler)
	c.On("command:removefromplaylist", removeFromPlaylistHandler)
	c.On("command:plistinfo", playlistInfoHandler)
	c.On("command:playlistinfo", playlistInfoHandler)
	c.On("command:myplist", myPlaylistsHandler)
	c.On("command:myplaylists", myPlaylistsHandler)

	c.On("callback:play_\\w+", playCallbackHandler, tg.CustomCallback(adminModeCB))
	c.On("callback:vcplay_\\w+", vcPlayHandler)
	c.On("callback:help_\\w+", helpCallbackHandler)
	c.On("callback:settings_\\w+", settingsCallbackHandler)
	c.On("callback:setlang_\\w+", setLangCallbackHandler)
	c.On("callback:fsub_\\w+", fsubCallbackHandler)

	c.AddParticipantHandler(handleParticipant)
	c.AddActionHandler(handleVoiceChatMessage)
	logger.Debug("Handlers loaded successfully.")
}
