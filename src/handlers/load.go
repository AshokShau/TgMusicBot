/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"log/slog"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/handlers"
	"github.com/AshokShau/gotdbot/handlers/filters/callbackquery"
)

var startTime = time.Now()

// LoadModules loads all the handlers.
// It takes a telegram gotdbot.Dispatcher as input.
func LoadModules(d *gotdbot.Dispatcher) {
	d.AddHandler(handlers.NewCommand("reload", reloadAdminCacheHandler))
	d.AddHandler(handlers.NewCommand("authList", authListHandler))
	d.AddHandler(handlers.NewCommand("auths", authListHandler))
	d.AddHandler(handlers.NewCommand("auth", addAuthHandler))
	d.AddHandler(handlers.NewCommand("addAuth", addAuthHandler))
	d.AddHandler(handlers.NewCommand("removeAuth", removeAuthHandler))
	d.AddHandler(handlers.NewCommand("rmAuth", removeAuthHandler))
	d.AddHandler(handlers.NewCommand("broadcast", broadcastHandler))
	d.AddHandler(handlers.NewCommand("gCast", broadcastHandler))
	d.AddHandler(handlers.NewCommand("cancelBroadcast", cancelBroadcastHandler))
	d.AddHandler(handlers.NewCommand("cancel", cancelBroadcastHandler))
	d.AddHandler(handlers.NewCommand("av", activeVcHandler))
	d.AddHandler(handlers.NewCommand("active_vc", activeVcHandler))
	d.AddHandler(handlers.NewCommand("clearass", clearAssistantsHandler))
	d.AddHandler(handlers.NewCommand("clearAssistants", clearAssistantsHandler))
	d.AddHandler(handlers.NewCommand("leaveAll", leaveAllHandler))
	d.AddHandler(handlers.NewCommand("logger", loggerHandler))
	d.AddHandler(handlers.NewCommand("privacy", privacyHandler))
	d.AddHandler(handlers.NewCommand("loop", loopHandler))
	d.AddHandler(handlers.NewCommand("pause", pauseHandler))
	d.AddHandler(handlers.NewCommand("resume", resumeHandler))
	d.AddHandler(handlers.NewCommand("cplist", createPlaylistHandler))
	d.AddHandler(handlers.NewCommand("createplaylist", createPlaylistHandler))
	d.AddHandler(handlers.NewCommand("deleteplaylist", deletePlaylistHandler))
	d.AddHandler(handlers.NewCommand("queue", queueHandler))
	d.AddHandler(handlers.NewCommand("seek", seekHandler))
	d.AddHandler(handlers.NewCommand("sh", shellCommand))
	d.AddHandler(handlers.NewCommand("skip", skipHandler))
	d.AddHandler(handlers.NewCommand("speed", speedHandler))
	d.AddHandler(handlers.NewCommand("stop", stopHandler))
	d.AddHandler(handlers.NewCommand("end", stopHandler))
	d.AddHandler(handlers.NewCommand("start", startHandler))
	d.AddHandler(handlers.NewCommand("ping", pingHandler))
	d.AddHandler(handlers.NewCommand("play", playHandler))
	d.AddHandler(handlers.NewCommand("p", playHandler))
	d.AddHandler(handlers.NewCommand("vplay", vPlayHandler))
	d.AddHandler(handlers.NewCommand("v", vPlayHandler))
	d.AddHandler(handlers.NewCommand("remove", removeHandler))
	d.AddHandler(handlers.NewCommand("mute", muteHandler))
	d.AddHandler(handlers.NewCommand("unmute", unmuteHandler))
	d.AddHandler(handlers.NewCommand("settings", settingsHandler))
	d.AddHandler(handlers.NewCommand("addtoplaylist", addToPlaylistHandler))
	d.AddHandler(handlers.NewCommand("addtoplist", addToPlaylistHandler))
	d.AddHandler(handlers.NewCommand("removefromplaylist", removeFromPlaylistHandler))
	d.AddHandler(handlers.NewCommand("rmplist", removeFromPlaylistHandler))
	d.AddHandler(handlers.NewCommand("plistinfo", playlistInfoHandler))
	d.AddHandler(handlers.NewCommand("playlistinfo", playlistInfoHandler))
	d.AddHandler(handlers.NewCommand("myplaylists", myPlaylistsHandler))
	d.AddHandler(handlers.NewCommand("myplist", myPlaylistsHandler))
	d.AddHandler(handlers.NewCommand("stats", statsHandler))

	d.AddHandler(handlers.NewUpdateNewCallbackQuery(callbackquery.Prefix("help_"), helpCallbackHandler))
	d.AddHandler(handlers.NewUpdateNewCallbackQuery(callbackquery.Prefix("play_"), playCallbackHandler))
	d.AddHandler(handlers.NewUpdateNewCallbackQuery(callbackquery.Prefix("vcplay_"), vcPlayHandler))
	d.AddHandler(handlers.NewUpdateNewCallbackQuery(callbackquery.Prefix("settings_"), settingsCallbackHandler))

	d.AddHandler(handlers.NewUpdateChatMember(nil, handleParticipant))
	d.AddHandler(handlers.NewUpdateNewMessage(nil, handleVoiceChatMessage))

	slog.Debug("Handlers loaded successfully")
}
