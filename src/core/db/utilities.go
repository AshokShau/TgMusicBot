/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var ctxBg = context.Background()

// toKey converts an int64 ID into a string format suitable for use as a cache key.
func toKey(id int64) string {
	return fmt.Sprintf("%d", id)
}

// getIntSlice safely converts an interface value into a slice of int64.
// It handles various numeric types and returns a boolean indicating the success of the conversion.
func getIntSlice(v interface{}) ([]int64, bool) {
	if v == nil {
		return []int64{}, false
	}

	switch val := v.(type) {
	case []int64:
		return val, true
	case []interface{}:
		return convertInterfaceSlice(val)
	case bson.A:
		return convertInterfaceSlice(val)
	default:
		log.Printf("Unexpected type encountered in getIntSlice: %T", v)
		return []int64{}, false
	}
}

// convertInterfaceSlice converts a slice of interfaces to a slice of int64
func convertInterfaceSlice(arr []interface{}) ([]int64, bool) {
	var out []int64
	for _, i := range arr {
		switch n := i.(type) {
		case int:
			out = append(out, int64(n))
		case int32:
			out = append(out, int64(n))
		case int64:
			out = append(out, n)
		case float64:
			if n == float64(int64(n)) {
				out = append(out, int64(n))
			}
		default:
			log.Printf("Unhandled numeric type in convertInterfaceSlice: %T", n)
			return nil, false
		}
	}
	return out, true
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

// Ctx creates a new context with a default timeout of 5 seconds.
// It returns the context and a cancel function to release resources.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctxBg, 5*time.Second)
}
