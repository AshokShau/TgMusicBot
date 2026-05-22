/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

type mediaAllocation struct {
	remaining int
	cleanup   func()
}

func (ctx *Client) registerMediaAllocation(chatId int64, devices []StreamDevice, cleanup func()) {
	if cleanup == nil {
		return
	}
	if len(devices) == 0 {
		cleanup()
		return
	}
	alloc := &mediaAllocation{
		remaining: len(devices),
		cleanup:   cleanup,
	}
	ctx.mediaAllocMu.Lock()
	defer ctx.mediaAllocMu.Unlock()
	if ctx.mediaAllocations == nil {
		ctx.mediaAllocations = make(map[int64]map[StreamDevice]*mediaAllocation)
	}
	deviceMap := ctx.mediaAllocations[chatId]
	if deviceMap == nil {
		deviceMap = make(map[StreamDevice]*mediaAllocation)
	}
	for _, device := range devices {
		if existing := deviceMap[device]; existing != nil {
			delete(deviceMap, device)
			existing.remaining--
			if existing.remaining == 0 {
				existing.cleanup()
			}
		}
		deviceMap[device] = alloc
	}
	ctx.mediaAllocations[chatId] = deviceMap
}

func (ctx *Client) releaseMediaAllocation(chatId int64, device StreamDevice) {
	ctx.mediaAllocMu.Lock()
	defer ctx.mediaAllocMu.Unlock()
	deviceMap := ctx.mediaAllocations[chatId]
	if deviceMap == nil {
		return
	}
	alloc := deviceMap[device]
	if alloc == nil {
		return
	}
	delete(deviceMap, device)
	alloc.remaining--
	if alloc.remaining == 0 {
		alloc.cleanup()
	}
	if len(deviceMap) == 0 {
		delete(ctx.mediaAllocations, chatId)
	}
}
