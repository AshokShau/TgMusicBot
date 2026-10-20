/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package bot

import (
	"ashokshau/tgmusic/internal/messaging"
	"time"

	"github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
)

var startTime = time.Now()

func LoadModules(c *gotdbot.Client, cancelCleanup func(int64)) {
	register := func(name string, handler func(*gotdbot.Client, *gotdbot.Message) error) {
		c.OnCommand(name, handler)
		if messaging.SupportsSilence(name) {
			c.OnCommand("s"+name, handler)
		}
	}
	c.OnCommand("reload", reloadAdminCacheHandler)
	c.OnCommand("join", joinHandler)
	c.OnCommand("link", joinHandler)
	c.OnCommand("authList", authListHandler)
	c.OnCommand("auths", authListHandler)
	c.OnCommand("auth", addAuthHandler)
	c.OnCommand("addAuth", addAuthHandler)
	c.OnCommand("removeAuth", removeAuthHandler)
	c.OnCommand("rmAuth", removeAuthHandler)
	c.OnCommand("broadcast", broadcastHandler)
	c.OnCommand("gCast", broadcastHandler)
	c.OnCommand("stop_gcast", cancelBroadcastHandler)
	c.OnCommand("stop_broadcast", cancelBroadcastHandler)
	c.OnCommand("av", activeVcHandler)
	c.OnCommand("active_vc", activeVcHandler)
	c.OnCommand("clearass", clearAssistantsHandler)
	c.OnCommand("clearAssistants", clearAssistantsHandler)
	c.OnCommand("leaveAll", leaveAllHandler)
	c.OnCommand("logger", loggerHandler)
	c.OnCommand("privacy", privacyHandler)
	c.OnCommand("autoplay", autoplayHandler)
	register("loop", loopHandler)
	register("pause", pauseHandler)
	register("resume", resumeHandler)
	c.OnCommand("cplist", createPlaylistHandler)
	c.OnCommand("createplaylist", createPlaylistHandler)
	c.OnCommand("deleteplaylist", deletePlaylistHandler)
	register("queue", queueHandler)
	register("seek", seekHandler)
	c.OnCommand("sh", shellCommand)
	register("skip", skipHandler)
	register("stop", stopHandler)
	register("end", stopHandler)
	c.OnCommand("start", startHandler)
	c.OnCommand("help", startHandler)
	c.OnCommand("ping", pingHandler)
	register("mix", mixHandler)
	register("play", playHandler)
	register("p", playHandler)
	register("fplay", fPlayHandler)
	register("fp", fPlayHandler)
	register("vplay", vPlayHandler)
	register("v", vPlayHandler)
	register("fvplay", fVPlayHandler)
	register("fvp", fVPlayHandler)
	register("remove", removeHandler)
	register("mute", muteHandler)
	register("unmute", unmuteHandler)
	c.OnCommand("settings", settingsHandler)
	c.OnCommand("addtoplaylist", addToPlaylistHandler)
	c.OnCommand("addtoplist", addToPlaylistHandler)
	c.OnCommand("removefromplaylist", removeFromPlaylistHandler)
	c.OnCommand("rmplist", removeFromPlaylistHandler)
	c.OnCommand("plistinfo", playlistInfoHandler)
	c.OnCommand("playlistinfo", playlistInfoHandler)
	c.OnCommand("myplaylists", myPlaylistsHandler)
	c.OnCommand("myplist", myPlaylistsHandler)
	c.OnCommand("stats", statsHandler)

	c.OnUpdateNewCallbackQuery(helpCallbackHandler, callbackquery.Prefix("help_"))
	c.OnUpdateNewCallbackQuery(playCallbackHandler, callbackquery.Prefix("play_"))
	c.OnUpdateNewCallbackQuery(vcPlayHandler, callbackquery.Prefix("vcplay_"))
	c.OnUpdateNewCallbackQuery(func(client *gotdbot.Client, cb *gotdbot.UpdateNewCallbackQuery) error {
		return settingsCallbackHandler(client, cb, cancelCleanup)
	}, callbackquery.Prefix("settings_"))
	c.OnUpdateNewCallbackQuery(autoplayCallbackHandler, callbackquery.Equal("autoplay_toggle"))

	c.OnUpdateChatMember(handleParticipant, nil)
	c.OnUpdateNewChat(handleNewChat, nil)
	c.OnUpdateNewMessage(handleVoiceChatMessage, nil)

	c.Logger.Debug("Handlers loaded successfully")
}
