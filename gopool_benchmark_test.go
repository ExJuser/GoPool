package GoPool

import (
	"sync"
	"testing"
	"time"
)

const (
	RunTimes           = 1000000
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
func BenchmarkGoroutines(b *testing.B) {
	wg := sync.WaitGroup{}
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			go func() {
				demoFunc()
				if j%1000 == 0 {
					b.Logf("Goroutine %d finished.", j)
				}
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkGoPool(b *testing.B) {
	wg := sync.WaitGroup{}
	pool, _ := NewPool(BenchPoolSize, WithExpiryDuration(DefaultExpiredTime))
	defer pool.Release()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(RunTimes)
		for j := 0; j < RunTimes; j++ {
			_ = pool.Submit(func() {
				demoFunc()
				if j%1000 == 0 {
					b.Logf("GoPool %d finished.", j)
				}
				wg.Done()
			})
		}
		wg.Wait()
	}
	b.StopTimer()
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
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < RunTimes; j++ {
			_ = pool.Submit(demoFunc)
		}
	}
	b.StopTimer()
}
