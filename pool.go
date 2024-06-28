package GoPool

import (
	"GoPool/internal"
	"context"
	"sync"
	"sync/atomic"
	"time"
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

func (p *Pool) purgePeriodically(ctx context.Context) {
	heartbeat := time.NewTicker(p.options.ExpiryDuration)
	defer func() {
		heartbeat.Stop()
		atomic.StoreInt32(&p.heartbeatDone, 1)
	}()
	for {
		select {
		case <-heartbeat.C:
		case <-ctx.Done():
			return
		}
		if p.IsClosed() {
			break
		}
		p.lock.Lock()
		expiredWorkers := p.workers.retrieveExpiry(p.options.ExpiryDuration)
		p.lock.Unlock()
		for i := range expiredWorkers {
			expiredWorkers[i].task = nil
			expiredWorkers[i] = nil
		}
		if p.Running() == 0 || (p.Waiting() > 0 && p.Free() > 0) {
			p.cond.Broadcast()
		}
	}
}

func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}
	var w *goWorker
	if w = p.retrieveWorker(); w == nil {
		return ErrPoolOverload
	}
	w.task <- task
	return nil
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}
func (p *Pool) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}
func (p *Pool) Free() int {

	if c := p.Cap(); c < 0 {
		return -1
	} else {
		return c - p.Running()
	}
}
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *Pool) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

func (p *Pool) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}
func (p *Pool) addRunning(delta int) {
	atomic.AddInt32(&p.running, int32(delta))
}

func (p *Pool) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		atomic.StoreInt32(&p.heartbeatDone, 0)
		var ctx context.Context
		ctx, p.stopHeartBeat = context.WithCancel(context.Background())
		go p.purgePeriodically(ctx)
	}
}
