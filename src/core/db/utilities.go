/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"fmt"
)

// toKey converts an int64 ID into a string format suitable for use as a cache key.
func toKey(id int64) string {
	return fmt.Sprintf("%d", id)
}

// contains checks if a given int64 slice contains a specific ID.
// It returns true if the ID is found, and false otherwise.
func contains(list []int64, id int64) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}
