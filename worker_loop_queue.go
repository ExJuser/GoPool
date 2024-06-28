package GoPool

import "time"

type loopQueue struct {
	items            []*goWorker
	expiry           []*goWorker
	head, tail, size int
	isFull           bool
}

func newWorkerLoopQueue(size int) *loopQueue {
	return &loopQueue{
		items: make([]*goWorker, size),
		size:  size,
	}
}

func (l *loopQueue) len() int {
	if l.size == 0 {
		return 0
	}
	if l.head == l.tail {
		if l.isFull {
			return l.size
		}
		return 0
	}
	if l.tail > l.head {
		return l.tail - l.head
	}
	return l.size - l.head + l.tail
}

func (l *loopQueue) isEmpty() bool {
	return l.head == l.tail && !l.isFull
}

func (l *loopQueue) insert(worker *goWorker) error {
	if l.size == 0 {
		return errQueueIsReleased
	}
	if l.isFull {
		return errQueueIsFull
	}
	l.items[l.tail] = worker
	l.tail++
	if l.tail == l.size {
		l.tail = 0
	}
	if l.tail == l.head {
		l.isFull = true
	}
	return nil
}

func (l *loopQueue) detach() *goWorker {
	if l.isEmpty() {
		return nil
	}
	w := l.items[l.head]
	l.items[l.head] = nil
	l.head++
	if l.head == l.size {
		l.head = 0
	}
	l.isFull = false
	return w
}

func (l *loopQueue) retrieveExpiry(duration time.Duration) []*goWorker {
	//TODO implement me
	panic("implement me")
}

func (l *loopQueue) reset() {
	//TODO implement me
	panic("implement me")
}
