/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ubot

func stdRemove[T comparable](slice []T, val T) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}
