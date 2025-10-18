package handlers

import (
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
)

var startTime = time.Now()

// LoadModules loads all the handlers.
// It takes a telegram client as input.
func LoadModules(c *telegram.Client) {
	_, _ = c.UpdatesGetState()

	c.On("command:ping", pingHandler)
	c.On("command:start", startHandler)
	c.On("command:help", startHandler)
	c.On("command:reload", reloadAdminCacheHandler)
	c.On("command:privacy", privacyHandler)

	c.On("command:play", playHandler, telegram.FilterFunc(playMode))
	c.On("command:vPlay", vPlayHandler, telegram.FilterFunc(playMode))

	c.On("command:loop", loopHandler, telegram.FilterFunc(adminMode))
	c.On("command:remove", removeHandler, telegram.FilterFunc(adminMode))
	c.On("command:skip", skipHandler, telegram.FilterFunc(adminMode))
	c.On("command:stop", stopHandler, telegram.FilterFunc(adminMode))
	c.On("command:end", stopHandler, telegram.FilterFunc(adminMode))
	c.On("command:mute", muteHandler, telegram.FilterFunc(adminMode))
	c.On("command:unmute", unmuteHandler, telegram.FilterFunc(adminMode))
	c.On("command:pause", pauseHandler, telegram.FilterFunc(adminMode))
	c.On("command:resume", resumeHandler, telegram.FilterFunc(adminMode))
	c.On("command:queue", queueHandler, telegram.FilterFunc(adminMode))
	c.On("command:seek", seekHandler, telegram.FilterFunc(adminMode))
	c.On("command:speed", speedHandler, telegram.FilterFunc(adminMode))
	c.On("command:authList", authListHandler, telegram.FilterFunc(adminMode))
	c.On("command:addAuth", addAuthHandler, telegram.FilterFunc(adminMode))
	c.On("command:auth", addAuthHandler, telegram.FilterFunc(adminMode))
	c.On("command:removeAuth", removeAuthHandler, telegram.FilterFunc(adminMode))
	c.On("command:unAuth", removeAuthHandler, telegram.FilterFunc(adminMode))
	c.On("command:rmAuth", removeAuthHandler, telegram.FilterFunc(adminMode))

	c.On("command:active_vc", activeVcHandler, telegram.FilterFunc(isDev))
	c.On("command:av", activeVcHandler, telegram.FilterFunc(isDev))
	c.On("command:stats", sysStatsHandler, telegram.FilterFunc(isDev))

	c.On("command:settings", settingsHandler, telegram.FilterFunc(adminMode))
	c.On("callback:play_\\w+", playCallbackHandler, telegram.FilterFuncCallback(adminModeCB))
	c.On("callback:vcplay_\\w+", vcPlayHandler)
	c.On("callback:help_\\w+", helpCallbackHandler)
	c.On("callback:settings_\\w+", settingsCallbackHandler)

	c.On(telegram.OnParticipant, handleParticipant)
	c.AddRawHandler(&telegram.UpdateNewChannelMessage{}, handleVoiceChat)
	gologging.Debug("Handlers loaded successfully.")
}
