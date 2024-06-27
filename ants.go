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
	DefaultGoPoolSize        = math.MaxInt32
	DefaultCleanIntervalTime = time.Second
)
const (
	OPENED = iota
	CLOSED
)

var (
	ErrLackPoolFunc        = errors.New("must provide function for pool")
	ErrInvalidPoolExpiry   = errors.New("invalid expiry for pool")
	ErrPoolClosed          = errors.New("this pool has been closed")
	ErrPoolOverload        = errors.New("too many goroutines blocked on submit or NonBlocking is set")
	ErrInvalidPreAllocSize = errors.New("can not set up a negative capacity under PreAlloc mode")
	ErrTimeout             = errors.New("operation timed out")
	workerChanCap          = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}
	defaultLogger    = Logger(log.New(os.Stderr, "", log.LstdFlags))
	defaultGoPool, _ = NewPool(DefaultGoPoolSize)
)

type Logger interface {
	Printf(format string, args ...interface{})
}
