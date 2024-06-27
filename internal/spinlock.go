package internal

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type spinLock uint32

const maxBackoff = 16

func (l *spinLock) Lock() {
	backoff := 1
	for !atomic.CompareAndSwapUint32((*uint32)(l), 0, 1) {
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
		}
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func (l *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(l), 0)
}

func NewSpinLock() sync.Locker {
	return new(spinLock)
}
