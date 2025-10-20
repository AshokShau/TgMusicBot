/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

type RTCServer struct {
	ID                 int64
	Ipv4, Ipv6         string
	Username, Password string
	Port               int32
	Turn, Stun, Tcp    bool
	PeerTag            []byte
}
