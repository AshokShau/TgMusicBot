/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

type Client struct {
	ptr                         uintptr
	connectionChangeCallbacks   []ConnectionChangeCallback
	streamEndCallbacks          []StreamEndCallback
	upgradeCallbacks            []UpgradeCallback
	signalCallbacks             []SignalCallback
	frameCallbacks              []FrameCallback
	remoteSourceCallbacks       []RemoteSourceCallback
	broadcastTimestampCallbacks []BroadcastTimestampCallback
	broadcastPartCallbacks      []BroadcastPartCallback
}
