package GoPool

import (
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 10000000
	BenchParam         = 10
	BenchPoolSize      = 200000
	DefaultExpiredTime = 10 * time.Second
)

func demoFunc() {
	time.Sleep(time.Duration(BenchParam) * time.Millisecond)
}
func demoPoolFunc(args interface{}) {
	n := args.(int)
	time.Sleep(time.Duration(n) * time.Millisecond)
}

// 使用原生Goroutine
func BenchmarkGoroutines(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

// 使用GoPool 实际上在go的最新版本 速度上已经不如原生goroutine了
// 但是能够节省很大一部分资源并且限制并发数量提升系统稳定性
func BenchmarkGoPool(b *testing.B) {
	wg := sync.WaitGroup{}
	pool, _ := NewPool(BenchPoolSize, WithExpiryDuration(DefaultExpiredTime))
	defer pool.Release()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = pool.Submit(func() {
				demoFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

func BenchmarkGoroutinesThroughput(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			go demoFunc()
		}
	}
}

func BenchmarkGoPoolThroughput(b *testing.B) {
	pool, _ := NewPool(DefaultGoPoolSize, WithExpiryDuration(DefaultExpiredTime))
	defer pool.Release()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = pool.Submit(demoFunc)
		}
	}
}
