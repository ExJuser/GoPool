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
	if size <= 0 { //负数size均认为是容量无上限
		size = -1
	}

	//过期时间不能为负
	if expiry := opts.ExpiryDuration; expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	} else if expiry == 0 { //使用默认的过期时间
		opts.ExpiryDuration = DefaultExpiryDuration
	}

	if opts.Logger == nil { //使用默认Logger
		opts.Logger = defaultLogger
	}

	p := &Pool{
		capacity: int32(size),
		lock:     internal.NewSpinLockBackoff(), //默认使用带指数退避的自旋锁
		//可选参数作为字段传入
		options: opts,
	}

	//sync.Pool中无Worker可取时的创建Worker方法
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
	} else { //不使用预分配模式则默认使用栈实现
		p.workers = newWorkerArray(stackType, 0)
	}

	p.cond = sync.NewCond(p.lock)

	//启动一个守护协程周期性的清理过期的worker直到心跳停止
	var ctx context.Context
	ctx, p.stopHeartBeat = context.WithCancel(context.Background())
	go p.purgePeriodically(ctx)

	return p, nil
}

func (p *Pool) purgePeriodically(ctx context.Context) {
	//启动一个每 ExpiryDuration 触发一次的定时器实现定时清扫过期Worker
	heartbeat := time.NewTicker(p.options.ExpiryDuration)
	defer func() {
		heartbeat.Stop()
		atomic.StoreInt32(&p.heartbeatDone, 1)
	}()

	for {
		select {
		//每隔p.options.ExpiryDuration时间启动一次清理过程
		case <-heartbeat.C:
		case <-ctx.Done():
			return
		}
		//case <-heartbeat.C触发后走到这里
		if p.IsClosed() {
			break
		}
		p.lock.Lock()
		//去除已经有p.options.ExpiryDuration未被使用的worker
		expiredWorkers := p.workers.retrieveExpiry(p.options.ExpiryDuration)
		p.lock.Unlock()

		//遍历这些过期worker并清理
		for i := range expiredWorkers {
			//传入nil任务 => worker.go:run():defer
			expiredWorkers[i].task <- nil
			expiredWorkers[i] = nil
		}

		//p.Running() == 0: 如果所有的worker都被过期清理
		if p.Running() == 0 || (p.Waiting() > 0 && p.Free() > 0) {
			p.cond.Broadcast()
		}
	}
}

func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		//向已经关闭的协程池提交任务
		return ErrPoolClosed
	}
	var w *goWorker
	//获取一个worker
	if w = p.retrieveWorker(); w == nil { //协程池已满
		return ErrPoolOverload
	}
	//并将任务塞给它
	w.task <- task
	return nil
}

// Running 原子操作获取当前正在运行的worker
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Waiting 原子操作获取当前正在等待的goroutine
func (p *Pool) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}

// Cap 原子操作获取协程池容量
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// Free 原子操作获取空闲（可能已经被回收 指还能多存在多少个worker）的worker
func (p *Pool) Free() int {
	var c int
	if c = p.Cap(); c < 0 {
		return -1
	}
	return c - p.Running()
}

// Tune 动态调整协程池容量
func (p *Pool) Tune(size int) {
	capacity := p.Cap()
	/**
	在以下情况下无需操作：
		1.容量已经为负（无上限） 不允许将容量调小
		2.size与原容量相同
		3.使用了预分配模式（循环队列），因此不允许调整
		4.size <= 0 为不合法操作
	*/
	if capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}
	//更新新的容量
	atomic.StoreInt32(&p.capacity, int32(size))
	//容量变大需要唤醒正在等待的goroutine
	if p.Waiting() > 0 && size > capacity {
		if size-capacity == 1 { //只有一个空闲
			p.cond.Signal()
			return
		}
		p.cond.Broadcast()
	}
}

// Release 关闭协程池并清空worker队列
func (p *Pool) Release() {
	//标记协程池关闭
	if !atomic.CompareAndSwapInt32(&p.state, OPENED, CLOSED) {
		return
	}
	//清空worker队列
	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()
	//如果还有正在等待的worker 将其唤醒
	p.cond.Broadcast()
}

func (p *Pool) ReleaseTimeout(timeout time.Duration) error {
	if p.IsClosed() || p.stopHeartBeat == nil {
		return ErrPoolClosed
	}
	p.stopHeartBeat()
	p.stopHeartBeat = nil
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

func (p *Pool) retrieveWorker() (w *goWorker) {
	spawnWorker := func() {
		//从sync.Pool中获得一个worker
		//可能会调用new方法新建一个 也可能是之前缓存的
		w = p.workerCache.Get().(*goWorker)
		//得到worker立刻将其启动
		w.run()
	}

	//对worker队列的操作都需要加锁
	p.lock.Lock()

	w = p.workers.detach() //尝试获取一个worker
	if w != nil {          //成功获得 直接解锁返回
		p.lock.Unlock()
	} else if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		//如果容量无限或容量还没有满 可以调用spawnWorker获取一个
		p.lock.Unlock()
		spawnWorker()
	} else { //容量满了
		if p.options.NonBlocking { //如果是非阻塞式的 不加入cond阻塞直接返回
			p.lock.Unlock()
			return
		}
	retry:
		//到这里意味着容量已满而且协程池是阻塞式的 需要阻塞等待
		//但是如果当前正在等待的协程数量已经超过了限额 直接返回
		if p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks {
			p.lock.Unlock()
			return
		}
		//加入阻塞等待
		p.addWaiting(1)
		p.cond.Wait()
		//到这一行就意味着已经被signal或broadcast唤醒
		p.addWaiting(-1)
		//关闭协程池也会唤醒被阻塞的worker
		if p.IsClosed() {
			p.lock.Unlock()
			return
		}
		var nw int
		//如果所有的worker都被过期清理了
		if nw = p.Running(); nw == 0 {
			p.lock.Unlock()
			//获取一个worker
			spawnWorker()
			return
		}
		//再次尝试获取一个worker 如果还是没有成功
		if w = p.workers.detach(); w == nil {
			if nw < p.Cap() { //还没有到达容量上限
				p.lock.Unlock()
				spawnWorker()
				return
			}
			//否则继续尝试获取
			goto retry
		}
		p.lock.Unlock()
	}
	return
}

// revertWorker 将使用完成的worker放回worker队列
func (p *Pool) revertWorker(worker *goWorker) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		p.cond.Broadcast()
		return false
	}
	//记录worker复用时间 长时间没有使用的worker会被回收
	worker.recycleTime = time.Now()
	p.lock.Lock()
	if p.IsClosed() {
		p.lock.Unlock()
		return false
	}
	if err := p.workers.insert(worker); err != nil {
		p.lock.Unlock()
		return false
	}
	p.cond.Signal()
	p.lock.Unlock()
	return true
}
