package GoPool

import (
	"errors"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)

const (
	// DefaultGoPoolSize 默认协程池容量
	DefaultGoPoolSize = math.MaxInt32
	// DefaultExpiryDuration 默认协程清理的间隔
	DefaultExpiryDuration = time.Second
)

// 指示协程池的开启关闭状态
const (
	OPENED = iota
	CLOSED
)

// 协程池的一些错误类型
var (
	//ErrLackPoolFunc 没有提供待执行的函数
	ErrLackPoolFunc = errors.New("must provide function for pool")

	//ErrInvalidPoolExpiry 过期时间为负
	ErrInvalidPoolExpiry = errors.New("invalid expiry for pool")

	//ErrPoolClosed 向已经关闭的协程池提交任务
	ErrPoolClosed = errors.New("this pool has been closed")

	//ErrPoolOverload  协程池已满
	ErrPoolOverload = errors.New("too many goroutines blocked on submit or NonBlocking is set")

	//ErrInvalidPreAllocSize 预分配空间模式下使用循环队列，不允许容量无上限
	ErrInvalidPreAllocSize = errors.New("can not set up a negative capacity under PreAlloc mode")

	//ErrTimeout 操作超时
	ErrTimeout = errors.New("operation timed out")
)

var (
	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}

	defaultLogger = Logger(log.New(os.Stderr, "", log.LstdFlags))

	//默认提供的协程池
	defaultGoPool, _ = NewPool(DefaultGoPoolSize)
)

type Logger interface {
	Printf(format string, args ...interface{})
}

//————————————————————————————————————————————
//可以直接通过包名访问的方法 操作的是默认提供的协程池

func Submit(task func()) error {
	return defaultGoPool.Submit(task)
}
func Running() int {
	return defaultGoPool.Running()
}
func Cap() int {
	return defaultGoPool.Cap()
}
func Free() int {
	return defaultGoPool.Free()
}
func Release() {
	defaultGoPool.Release()
}
func Reboot() {
	defaultGoPool.Reboot()
}
