/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"context"
	"fmt"
	"time"
)

var ctxBg = context.Background()

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

// remove creates a new slice that excludes a specific ID from the original int64 slice.
func remove(list []int64, id int64) []int64 {
	var newList []int64
	for _, v := range list {
		if v != id {
			newList = append(newList, v)
		}
	}
	return newList
}

// Ctx creates a new context with a default timeout of 10 seconds.
// It returns the context and a cancel function to release resources.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctxBg, 10*time.Second)
}
