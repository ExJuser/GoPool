package GoPool

import (
	"runtime"
	"time"
)

type goWorker struct {
	//回指所属协程池的指针 找到回家的路
	pool *Pool
	//这个worker应该完成的任务
	task chan func()
	//回收时间 当一个worker被放回协程池后更新
	//用于定期清理过期的worker：长时间未被使用
	recycleTime time.Time
}

func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		defer func() {
			w.pool.addRunning(-1)
			//将worker放回sync.Pool
			w.pool.workerCache.Put(w)
			//捕获run过程中出现的panic
			if p := recover(); p != nil {
				//调用自定义的PanicHandler
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker exits from panic: %s\n", string(buf[:n]))
				}
			}
			//通知唤醒一个向协程池提交任务，因为没有可用worker而陷入阻塞的goroutine
			w.pool.cond.Signal()
		}()
		//不断遍历channel 等待新投递过来的任务
		for f := range w.task {
			//投递一个nil的任务意味着该worker任务完成 可以被放回workerCache回收/复用
			if f == nil {
				return
			}
			f()
			//执行完一个任务之后就放回 等待Submit的新任务
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
