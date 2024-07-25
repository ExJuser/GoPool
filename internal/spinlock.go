package internal

import (
	"runtime"
	"sync"
	"sync/atomic"
)

const maxBackoff = 16 // 最大退避次数

type (
	spinLockBackoff uint32
	spinLock        uint32
)

func (l *spinLockBackoff) Lock() {
	backoff := 1
	// 尝试使用原子操作将锁从0设置为1，表示加锁成功
	for !atomic.CompareAndSwapUint32((*uint32)(l), 0, 1) {
		// 如果加锁失败，进行退避等待
		for i := 0; i < backoff; i++ {
			runtime.Gosched() // 让出 CPU 时间片，避免忙等
		}
		//不超过最大退避次数
		if backoff < maxBackoff {
			//指数退避
			backoff <<= 1
		}
	}
}

func (l *spinLockBackoff) Unlock() {
	atomic.StoreUint32((*uint32)(l), 0) //解锁只需原子操作将锁置为0即可
}

func (l *spinLock) Lock() {
	for !atomic.CompareAndSwapUint32((*uint32)(l), 0, 1) {
		runtime.Gosched()
	}
}

func (l *spinLock) Unlock() {
	atomic.StoreUint32((*uint32)(l), 0)
}

func NewSpinLockBackoff() sync.Locker {
	return new(spinLockBackoff)
}

func NewSpinLockOrigin() sync.Locker {
	return new(spinLock)
}
