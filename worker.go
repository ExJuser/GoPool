package GoPool

import (
	"runtime"
	"time"
)

type goWorker struct {
	pool        *Pool
	task        chan func()
	recycleTime time.Time
}

func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		defer func() {
			w.pool.addRunning(-1)
			w.pool.workerCache.Put(w)
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker exits from panic: %s\n", string(buf[:n]))
				}
			}
			w.pool.cond.Signal()
		}()
		for f := range w.task {
			if f == nil {
				return
			}
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
