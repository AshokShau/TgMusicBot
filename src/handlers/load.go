/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
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
	c.On("command:reload", reloadAdminCacheHandler)
	c.On("command:privacy", privacyHandler)
	c.On("command:setRtmp ", setRtmpHandler)

	c.On("command:play", playHandler, tg.CustomFilter(playMode))
	c.On("command:vPlay", vPlayHandler, tg.CustomFilter(playMode))
	c.On("command:stream", streamHandler, tg.CustomFilter(playMode))

	c.On("command:stopStream", stopStreamHandler, tg.CustomFilter(adminMode))
	c.On("command:loop", loopHandler, tg.CustomFilter(adminMode))
	c.On("command:remove", removeHandler, tg.CustomFilter(adminMode))
	c.On("command:skip", skipHandler, tg.CustomFilter(adminMode))
	c.On("command:stop", stopHandler, tg.CustomFilter(adminMode))
	c.On("command:end", stopHandler, tg.CustomFilter(adminMode))
	c.On("command:mute", muteHandler, tg.CustomFilter(adminMode))
	c.On("command:unmute", unmuteHandler, tg.CustomFilter(adminMode))
	c.On("command:pause", pauseHandler, tg.CustomFilter(adminMode))
	c.On("command:resume", resumeHandler, tg.CustomFilter(adminMode))
	c.On("command:queue", queueHandler, tg.CustomFilter(adminMode))
	c.On("command:seek", seekHandler, tg.CustomFilter(adminMode))
	c.On("command:speed", speedHandler, tg.CustomFilter(adminMode))
	c.On("command:authList", authListHandler, tg.CustomFilter(adminMode))
	c.On("command:addAuth", addAuthHandler, tg.CustomFilter(adminMode))
	c.On("command:auth", addAuthHandler, tg.CustomFilter(adminMode))
	c.On("command:removeAuth", removeAuthHandler, tg.CustomFilter(adminMode))
	c.On("command:unAuth", removeAuthHandler, tg.CustomFilter(adminMode))
	c.On("command:rmAuth", removeAuthHandler, tg.CustomFilter(adminMode))

	c.On("command:active_vc", activeVcHandler, tg.CustomFilter(isDev))
	c.On("command:av", activeVcHandler, tg.CustomFilter(isDev))
	c.On("command:stats", sysStatsHandler, tg.CustomFilter(isDev))
	c.On("command:streams", listStreamsHandler, tg.CustomFilter(isDev))
	c.On("command:clear_assistants", clearAssistantsHandler, tg.CustomFilter(isDev))
	c.On("command:clearAss", clearAssistantsHandler, tg.CustomFilter(isDev))
	c.On("command:leaveAll", leaveAllHandler, tg.CustomFilter(isDev))
	c.On("command:logger", loggerHandler, tg.CustomFilter(isDev))
	c.On("command:broadcast", broadcastHandler, tg.CustomFilter(isDev))
	c.On("command:gCast", broadcastHandler, tg.CustomFilter(isDev))
	c.On("command:cancelBroadcast", cancelBroadcastHandler, tg.CustomFilter(isDev))
	c.On("command:sh", shellCommand, tg.CustomFilter(isDev))

	c.On("command:settings", settingsHandler, tg.CustomFilter(adminMode))

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

	c.AddParticipantHandler(handleParticipant)
	c.AddActionHandler(handleVoiceChatMessage)
	logger.Debug("Handlers loaded successfully.")
}
