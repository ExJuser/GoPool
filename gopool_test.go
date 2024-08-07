package GoPool

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"sync"
	"sync/atomic"
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

func longRunningFunc() {
	for {
		runtime.Gosched()
	}
}
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

func TestGoPoolWithFuncGetWorkerFromCache(t *testing.T) {
	dur := 10
	p, _ := NewPoolWithFunc(TestSize, demoPoolFunc)
	defer p.Release()

	for i := 0; i < PoolSize; i++ {
		_ = p.Invoke(dur)
	}
	time.Sleep(2 * DefaultExpiryDuration)
	_ = p.Invoke(dur)
	t.Logf("num of current running workers:%d", p.running)
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestGoPoolWithFuncGetWorkerFromCacheInPreAllocMode(t *testing.T) {
	dur := 10
	p, _ := NewPoolWithFunc(TestSize, demoPoolFunc, WithPreAlloc(true))
	defer p.Release()

	for i := 0; i < PoolSize; i++ {
		_ = p.Invoke(dur)
	}
	time.Sleep(2 * DefaultExpiryDuration)
	_ = p.Invoke(dur)
	t.Logf("num of current running workers:%d", p.running)
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestNativeGoroutines(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			demoFunc()
			wg.Done()
		}()
	}

	wg.Wait()
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestGoPool(t *testing.T) {
	defer Release()
	wg := sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		_ = Submit(func() {
			demoFunc()
			wg.Done()
		})
	}
	wg.Wait()

	t.Logf("Capacity of the Pool:%d", Cap())
	t.Logf("Num of running workers:%d", Running())
	t.Logf("Num of free workers:%d", Free())
	mem := runtime.MemStats{}
	runtime.ReadMemStats(&mem)
	curMem = mem.TotalAlloc/MiB - curMem
	t.Logf("memory usage:%d MB", curMem)
}

func TestPanicHandler(t *testing.T) {
	var panicCount int64
	wg := sync.WaitGroup{}
	p, err := NewPool(10, WithPanicHandler(func(i interface{}) {
		defer wg.Done()
		atomic.AddInt64(&panicCount, 1)
		t.Logf("catch panic with panic handler:%v", i)
	}))
	assert.NoErrorf(t, err, "new pool failed:%v", err)
	defer p.Release()
	wg.Add(1)
	_ = p.Submit(func() {
		panic("oops")
	})
	wg.Wait()
	c := atomic.LoadInt64(&panicCount)
	assert.EqualValuesf(t, 1, c, "panic handler misfunctions")
	assert.EqualValuesf(t, 0, p.Running(), "there should be no worker running after panic")
}

func TestPanicHandlerWithFunc(t *testing.T) {
	var panicCount int64
	wg := sync.WaitGroup{}
	p, _ := NewPoolWithFunc(10, demoPoolFunc, WithPanicHandler(func(i interface{}) {
		defer wg.Done()
		atomic.AddInt64(&panicCount, 1)
		t.Logf("catch panic with panic handler:%v", i)
	}))
	defer p.Release()
	wg.Add(1)
	_ = p.Invoke("oops")
	wg.Wait()
	c := atomic.LoadInt64(&panicCount)
	assert.EqualValuesf(t, 1, c, "panic handler misfunctions")
	assert.EqualValuesf(t, 0, p.Running(), "there should be no worker running after panic")
}
func TestPurge(t *testing.T) {
	p, err := NewPool(10)
	assert.NoErrorf(t, err, "create TimingPool failed: %v", err)
	defer p.Release()
	_ = p.Submit(demoFunc)
	time.Sleep(3 * DefaultExpiryDuration)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
	p1, err := NewPoolWithFunc(10, demoPoolFunc)
	assert.NoErrorf(t, err, "create TimingPoolWithFunc failed: %v", err)
	defer p1.Release()
	_ = p1.Invoke(1)
	time.Sleep(3 * DefaultExpiryDuration)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
}

func TestPurgePreMalloc(t *testing.T) {
	p, err := NewPool(10, WithPreAlloc(true))
	assert.NoErrorf(t, err, "create TimingPool failed: %v", err)
	defer p.Release()
	_ = p.Submit(demoFunc)
	time.Sleep(3 * DefaultExpiryDuration)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
	p1, err := NewPoolWithFunc(10, demoPoolFunc)
	assert.NoErrorf(t, err, "create TimingPoolWithFunc failed: %v", err)
	defer p1.Release()
	_ = p1.Invoke(1)
	time.Sleep(3 * DefaultExpiryDuration)
	assert.EqualValues(t, 0, p.Running(), "all p should be purged")
}

func TestNonblockingSubmit(t *testing.T) {
	poolSize := 10
	p, err := NewPool(poolSize, WithNonBlocking(true))
	assert.NoErrorf(t, err, "create TimingPool failed: %v", err)
	defer p.Release()
	for i := 0; i < poolSize-1; i++ {
		assert.NoError(t, p.Submit(longRunningFunc), "nonblocking submit when pool is not full shouldn't return error")
	}
	ch := make(chan struct{})
	ch1 := make(chan struct{})
	f := func() {
		<-ch
		close(ch1)
	}
	// p is full now.
	assert.NoError(t, p.Submit(f), "nonblocking submit when pool is not full shouldn't return error")
	assert.EqualError(t, p.Submit(demoFunc), ErrPoolOverload.Error(),
		"nonblocking submit when pool is full should get an ErrPoolOverload")
	// interrupt f to get an available worker
	close(ch)
	<-ch1
	assert.NoError(t, p.Submit(demoFunc), "nonblocking submit when pool is not full shouldn't return error")
}

func TestRebootDefaultPool(t *testing.T) {
	defer Release()
	Reboot()
	wg := sync.WaitGroup{}
	wg.Add(1)
	_ = Submit(func() {
		demoFunc()
		wg.Done()
	})
	wg.Wait()
	Release()
	assert.EqualError(t, Submit(nil), ErrPoolClosed.Error(), "pool should be closed")
	Reboot()
	wg.Add(1)
	assert.NoErrorf(t, Submit(func() {
		wg.Done()
	}), "pool should be rebooted")
	wg.Wait()
}
