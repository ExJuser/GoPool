package main

import (
	"GoPool"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	runTimes            = 10000
	size                = 1000
	wg                  = sync.WaitGroup{}
	concurrentSum int32 = 0
)

func demoFunc() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println("Hello World!")
}
func demoConcurrentAddFunc(i interface{}) {
	n := i.(int32)
	fmt.Println("running with ", n)
	atomic.AddInt32(&concurrentSum, n)
}
func main() {
	//common pool
	pool, err := GoPool.NewPool(size)
	if err != nil {
		panic(err)
	}
	defer pool.Release()
	for i := 0; i < runTimes; i++ {
		wg.Add(1)
		err = pool.Submit(func() {
			demoFunc()
			wg.Done()
		})
		if err != nil {
			panic(err)
		}
	}
	wg.Wait()
	fmt.Println("running goroutines:", pool.Running())
	fmt.Println("finish all tasks.")

	//pool with a func
	poolWithFunc, err := GoPool.NewPoolWithFunc(size, func(i interface{}) {
		demoConcurrentAddFunc(i)
		wg.Done()
	})
	if err != nil {
		panic(err)
	}
	for i := 0; i < runTimes; i++ {
		wg.Add(1)
		_ = poolWithFunc.Invoke(int32(i))
	}
	wg.Wait()
	fmt.Println("running goroutines:", pool.Running())
	fmt.Println("finish all tasks, the result is", concurrentSum)
}
