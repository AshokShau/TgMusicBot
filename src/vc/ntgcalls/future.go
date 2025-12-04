package ntgcalls

import (
	"runtime/cgo"
	"sync"
)

// #include "ntgcalls.h"
// #include <stdlib.h>
// extern void unlockMutex(void*);
import "C"
import (
	"unsafe"
)

type Future struct {
	mutex      *sync.Mutex
	errCode    *C.int
	errMessage **C.char
}

func CreateFuture() *Future {
	errCodePtr := (*C.int)(C.malloc(C.size_t(unsafe.Sizeof(C.int(0)))))
	errMessagePtr := (**C.char)(C.malloc(C.size_t(unsafe.Sizeof((*C.char)(nil)))))

	res := &Future{
		mutex:      &sync.Mutex{},
		errCode:    errCodePtr,
		errMessage: errMessagePtr,
	}
	res.mutex.Lock()
	return res
}

func (ctx *Future) ParseToC() C.ntg_async_struct {
	var x C.ntg_async_struct
	h := cgo.NewHandle(ctx.mutex)

	ptr := C.malloc(C.size_t(unsafe.Sizeof(h)))
	*(*cgo.Handle)(ptr) = h

	x.userData = ptr
	x.promise = (C.ntg_async_callback)(unsafe.Pointer(C.unlockMutex))
	x.errorCode = (*C.int)(unsafe.Pointer(ctx.errCode))
	x.errorMessage = ctx.errMessage
	return x
}

func (ctx *Future) wait() {
	ctx.mutex.Lock()
}

func (ctx *Future) Free() {
	if ctx.errCode != nil {
		C.free(unsafe.Pointer(ctx.errCode))
		ctx.errCode = nil
	}
	if ctx.errMessage != nil {
		C.free(unsafe.Pointer(ctx.errMessage))
		ctx.errMessage = nil
	}
}

//export unlockMutex
func unlockMutex(p unsafe.Pointer) {
	h := *(*cgo.Handle)(p)
	C.free(p)

	m := h.Value().(*sync.Mutex)
	m.Unlock()
	h.Delete()
}
