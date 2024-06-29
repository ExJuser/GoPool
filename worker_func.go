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
			w.pool.workerCache.Put(w)
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
			w.pool.cond.Signal()
		}()

		for args := range w.args {
			if args == nil {
				return
			}
			w.pool.poolFunc(args)
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
