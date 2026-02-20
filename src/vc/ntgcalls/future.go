/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package ntgcalls

import (
	"fmt"
	"sync"
	"time"
)

// #include "ntgcalls.h"
// extern void futureCallback(void*);
import "C"
import (
	"unsafe"
)

var (
	futuresMu sync.Mutex
	futures   = make(map[uintptr]*Future)
	futureID  uintptr
)

type Future struct {
	id         uintptr
	done       chan struct{}
	errCode    *C.int
	errMessage **C.char
}

func CreateFuture() *Future {
	f := &Future{
		done:       make(chan struct{}),
		errCode:    new(C.int),
		errMessage: new(*C.char),
	}

	futuresMu.Lock()
	futureID++
	f.id = futureID
	futures[f.id] = f
	futuresMu.Unlock()

	return f
}

func (ctx *Future) ParseToC() C.ntg_async_struct {
	var x C.ntg_async_struct
	x.userData = unsafe.Pointer(ctx.id)
	x.promise = (C.ntg_async_callback)(unsafe.Pointer(C.futureCallback))
	x.errorCode = (*C.int)(unsafe.Pointer(ctx.errCode))
	x.errorMessage = ctx.errMessage
	return x
}

func (ctx *Future) wait() error {
	select {
	case <-ctx.done:
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("ntgcalls timeout")
	}
}

//export futureCallback
func futureCallback(p unsafe.Pointer) {
	id := uintptr(p)

	futuresMu.Lock()
	defer futuresMu.Unlock()

	if f, ok := futures[id]; ok {
		close(f.done)
		delete(futures, id)
	}
}
