/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

type RemoteSource struct {
	Ssrc   uint32
	State  StreamStatus
	Device StreamDevice
}
