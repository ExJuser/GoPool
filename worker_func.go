package GoPool

import "time"

type goWorkerWithFunc struct {
	pool        *PoolWithFunc
	args        chan interface{}
	recycleTime time.Time
}

func (w *goWorkerWithFunc) run() {

}
