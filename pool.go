package GoPool

import (
	"GoPool/internal"
	"context"
	"sync"
	"sync/atomic"
)

type Pool struct {
	//协程池的容量，负值意味着容量无上限
	capacity int32
	//当前正在运行的协程数量
	running int32
	//保护并发访问的锁 采用动态退避的自旋锁
	lock sync.Locker
	//存储可用worker的集合
	workers     workerArray
	state       int32
	cond        *sync.Cond
	workerCache sync.Pool
	//被阻塞在等待可用worker的协程数量
	waiting       int32
	heartbeatDone int32
	stopHeartBeat context.CancelFunc
	options       *Options
}

func NewPool(size int, options ...Option) (*Pool, error) {
	opts := loadOptions(options...)
	if size <= 0 {
		size = -1
	}
	if expiry := opts.ExpiryDuration; expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	} else if expiry == 0 {
		opts.ExpiryDuration = DefaultCleanIntervalTime
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}
	p := &Pool{
		capacity: int32(size),
		lock:     internal.NewSpinLockBackoff(),
		options:  opts,
	}
	p.workerCache.New = func() any {
		return &goWorker{
			pool: p,
			task: make(chan func(), workerChanCap()),
		}
	}
	if p.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		p.workers = newWorkerArray(loopQueueType, size)
	} else {
		p.workers = newWorkerArray(stackType, 0)
	}
	p.cond = sync.NewCond(p.lock)

	//启动一个守护协程周期性的清理过期的worker
	var ctx context.Context
	ctx, p.stopHeartBeat = context.WithCancel(context.Background())
	go p.purgePeriodically(ctx)
	return p, nil
}

func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}
func (p *Pool) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *Pool) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}
func (p *Pool) addRunning(delta int) {
	atomic.AddInt32(&p.running, int32(delta))
}
