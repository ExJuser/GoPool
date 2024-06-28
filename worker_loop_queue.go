package GoPool

type loopQueue struct {
	items            []*goWorker
	expiry           []*goWorker
	head, tail, size int
	isFull           bool
}

func NewWorkerLoopQueue(size int) *loopQueue {
	return &loopQueue{
		items: make([]*goWorker, size),
		size:  size,
	}
}
