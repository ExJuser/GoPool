package GoPool

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

const (
	_   = 1 << (10 * iota)
	KiB // 1024
	MiB // 1048576
	// GiB // 1073741824
	// TiB // 1099511627776             (超过了int32的范围)
	// PiB // 1125899906842624
	// EiB // 1152921504606846976
	// ZiB // 1180591620717411303424    (超过了int64的范围)
	// YiB // 1208925819614629174706176
)
const (
	Param    = 100
	PoolSize = 1000
	TestSize = 10000
	n        = 100000
)

var curMem uint64

func TestGoPoolWaitingToRetrieveWorker(t *testing.T) {
	wg := sync.WaitGroup{}
	p, _ := NewPool(PoolSize)
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()
	t.Logf("num of current running workers:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestGoPoolWaitingToRetrieveWorkerInPreAllocMode(t *testing.T) {
	wg := sync.WaitGroup{}
	p, _ := NewPool(PoolSize, WithPreAlloc(true))
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			demoPoolFunc(Param)
			wg.Done()
		})
	}
	wg.Wait()
	t.Logf("num of current running workers:%d", p.running)
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d", curMem)
}

func TestGoPoolWithFuncWaitingToRetrieveWorker(t *testing.T) {
	wg := sync.WaitGroup{}
	p, _ := NewPoolWithFunc(PoolSize, func(i interface{}) {
		demoPoolFunc(i)
		wg.Done()
	})
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Invoke(Param)
	}
	wg.Wait()
	t.Logf("num of current running workers:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestGoPoolWithFuncWaitingToRetrieveWorkerInPreAllocMode(t *testing.T) {
	wg := sync.WaitGroup{}
	p, _ := NewPoolWithFunc(PoolSize, func(i interface{}) {
		demoPoolFunc(i)
		wg.Done()
	}, WithPreAlloc(true))
	defer p.Release()

	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = p.Invoke(Param)
	}
	wg.Wait()
	t.Logf("pool with func, running workers number:%d", p.Running())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestGoPoolGetWorkerFromCache(t *testing.T) {
	p, _ := NewPool(TestSize)
	defer p.Release()

	for i := 0; i < PoolSize; i++ {
		_ = p.Submit(demoFunc)
	}
	//前面提交的任务已经完成而且worker已经被超时清理放入cache
	time.Sleep(2 * DefaultExpiryDuration)
	_ = p.Submit(demoFunc)
	t.Logf("num of current running workers:%d", p.running)
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	//应该是1 从workerCache中取出
	t.Logf("memory usage:%d MB", curMem)
}
