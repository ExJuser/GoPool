package GoPool

import (
	"runtime"
	"time"
)

type goWorkerWithFunc struct {
	pool        *PoolWithFunc
	args        chan interface{}
	recycleTime time.Time
}

func (w *goWorkerWithFunc) run() {
	w.pool.addRunning(1)
	go func() {
		defer func() {
			w.pool.addRunning(-1)
			//将worker放回
			w.pool.workerCache.Put(w)
			//从执行任务的panic中恢复
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker with func exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker with func exits from panic: %s\n", string(buf[:n]))
				}
			}
			//唤醒一个阻塞的取worker失败的goroutine
			w.pool.cond.Signal()
		}()

		//只有args通道被关闭或者args传入nil才会结束轮询
		for args := range w.args {
			if args == nil {
				return //defer
			}
			w.pool.poolFunc(args)
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
