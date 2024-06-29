package GoPool

import (
	"GoPool/internal"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type PoolWithFunc struct {
	capacity      int32
	running       int32
	lock          sync.Locker
	workers       []*goWorkerWithFunc
	state         int32
	cond          *sync.Cond
	poolFunc      func(interface{})
	workerCache   sync.Pool
	waiting       int32
	heartbeatDone int32
	stopHeartbeat context.CancelFunc
	options       *Options
}

func (p *PoolWithFunc) purgePeriodically(ctx context.Context) {
	heartbeat := time.NewTicker(p.options.ExpiryDuration)
	defer func() {
		heartbeat.Stop()
		atomic.StoreInt32(&p.heartbeatDone, 1)
	}()

	var expiredWorkers []*goWorkerWithFunc
	for {
		select {
		case <-heartbeat.C:
		case <-ctx.Done():
			return
		}
		if p.IsClosed() {
			break
		}
		currentTime := time.Now()
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers)
		var i int
		for i = 0; i < n && currentTime.Sub(idleWorkers[i].recycleTime) > p.options.ExpiryDuration; i++ {
		}
		expiredWorkers = append(expiredWorkers[:0], idleWorkers[:i]...)
		if i > 0 {
			m := copy(idleWorkers, idleWorkers[i:])
			for i = m; i < n; i++ {
				idleWorkers[i] = nil
			}
			p.workers = idleWorkers[:m]
		}
		p.lock.Unlock()
		for i, w := range expiredWorkers {
			w.args <- nil
			expiredWorkers[i] = nil
		}
		if p.Running() == 0 || (p.Waiting() > 0 && p.Free() > 0) {
			p.cond.Broadcast()
		}
	}
}

func NewPoolWithFunc(size int, pf func(interface{}), options ...Option) (*PoolWithFunc, error) {
	if size <= 0 {
		size = -1
	}
	if pf == nil {
		return nil, ErrLackPoolFunc
	}
	opts := loadOptions(options...)
	if expiry := opts.ExpiryDuration; expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	} else if expiry == 0 {
		opts.ExpiryDuration = DefaultCleanIntervalTime
	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}
	p := &PoolWithFunc{
		capacity: int32(size),
		poolFunc: pf,
		lock:     internal.NewSpinLockBackoff(),
		options:  opts,
	}
	p.workerCache.New = func() any {
		return &goWorkerWithFunc{
			pool: p,
			args: make(chan interface{}, workerChanCap()),
		}
	}
	if p.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		p.workers = make([]*goWorkerWithFunc, 0, size)
	}
	p.cond = sync.NewCond(p.lock)

	var ctx context.Context
	ctx, p.stopHeartbeat = context.WithCancel(context.Background())
	go p.purgePeriodically(ctx)
	return p, nil
}
func (p *PoolWithFunc) Invoke(args interface{}) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}
	var w *goWorkerWithFunc
	if w = p.retrieveWorker(); w == nil {
		return ErrPoolOverload
	}
	w.args <- args
	return nil
}

func (p *PoolWithFunc) Running() int {
	return int(atomic.LoadInt32(&p.running))
}
func (p *PoolWithFunc) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}
func (p *PoolWithFunc) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}
func (p *PoolWithFunc) Free() int {
	var c int
	if c = p.Cap(); c < 0 {
		return -1
	}
	return c - p.Running()
}

func (p *PoolWithFunc) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

func (p *PoolWithFunc) addRunning(delta int) {
	atomic.AddInt32(&p.running, int32(delta))
}

func (p *PoolWithFunc) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}

func (p *PoolWithFunc) Reboot() {
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		atomic.StoreInt32(&p.heartbeatDone, 0)
		var ctx context.Context
		ctx, p.stopHeartbeat = context.WithCancel(context.Background())
		go p.purgePeriodically(ctx)
	}
}

func (p *PoolWithFunc) Tune(size int) {
	capacity := p.Cap()
	if capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}
	atomic.StoreInt32(&p.capacity, int32(size))
	if size > capacity {
		if size-capacity == 1 {
			p.cond.Signal()
			return
		}
		p.cond.Broadcast()
	}
}

func (p *PoolWithFunc) Release() {
	if !atomic.CompareAndSwapInt32(&p.state, OPENED, CLOSED) {
		return
	}
	p.lock.Lock()
	idleWorkers := p.workers
	for _, w := range idleWorkers {
		w.args <- nil
	}
	p.workers = nil
	p.lock.Unlock()
	p.cond.Broadcast()
}

func (p *PoolWithFunc) ReleaseTimeout(timeout time.Duration) error {
	if p.IsClosed() || p.stopHeartbeat == nil {
		return ErrPoolClosed
	}
	p.stopHeartbeat()
	p.stopHeartbeat = nil
	p.Release()

	endTime := time.Now().Add(timeout)
	for time.Now().Before(endTime) {
		if p.Running() == 0 && atomic.LoadInt32(&p.heartbeatDone) == 1 {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ErrTimeout
}

func (p *PoolWithFunc) retrieveWorker() (w *goWorkerWithFunc) {
	spawnWorker := func() {
		w = p.workerCache.Get().(*goWorkerWithFunc)
		w.run()
	}

	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n >= 0 {
		w = idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
		p.lock.Unlock()
	} else if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		p.lock.Unlock()
		spawnWorker()
	} else {
		if p.options.NonBlocking {
			p.lock.Unlock()
			return
		}
	retry:
		if p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks {
			p.lock.Unlock()
			return
		}
		p.addWaiting(1)
		p.cond.Wait()
		p.addWaiting(-1)

		if p.IsClosed() {
			p.lock.Unlock()
			return
		}

		var nw int
		if nw = p.Running(); nw == 0 {
			p.lock.Unlock()
			spawnWorker()
			return
		}
		l := len(p.workers) - 1
		if l < 0 {
			if nw < p.Cap() {
				p.lock.Unlock()
				spawnWorker()
				return
			}
			goto retry
		}
		w = p.workers[l]
		p.workers[l] = nil
		p.workers = p.workers[:l]
		p.lock.Unlock()
	}
	return
}

func (p *PoolWithFunc) revertWorker(worker *goWorkerWithFunc) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		p.cond.Broadcast()
		return false
	}
	worker.recycleTime = time.Now()
	p.lock.Lock()

	if p.IsClosed() {
		p.lock.Unlock()
		return false
	}

	p.workers = append(p.workers, worker)

	p.cond.Signal()
	p.lock.Unlock()
	return true
}
