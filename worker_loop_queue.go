package GoPool

import (
	"time"
)

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
	expiryTime := time.Now().Add(-duration)
	index := l.binarySearch(expiryTime)
	if index == -1 {
		return nil
	}
	l.expiry = l.expiry[:0]
	if l.head <= index {
		l.expiry = append(l.expiry, l.items[l.head:index+1]...)
		for i := l.head; i < index+1; i++ {
			l.items[i] = nil
		}
	} else {
		l.expiry = append(l.expiry, l.items[0:index+1]...)
		l.expiry = append(l.expiry, l.items[l.head:]...)
		for i := 0; i < index+1; i++ {
			l.items[i] = nil
		}
		for i := l.head; i < l.size; i++ {
			l.items[i] = nil
		}
	}
	head := (index + 1) % l.size
	l.head = head
	if len(l.expiry) > 0 {
		l.isFull = false
	}
	return l.expiry
}

func (l *loopQueue) binarySearch(expiryTime time.Time) int {
	var mid, nlen, basel, tmid int
	nlen = len(l.items)
	if l.isEmpty() || expiryTime.Before(l.items[l.head].recycleTime) {
		return -1
	}
	right := (l.tail - l.head + nlen - 1) % nlen
	basel = l.head
	left := 0
	for left <= right {
		mid = left + ((right - left) >> 1)
		tmid = (mid + basel + nlen) % nlen
		if expiryTime.Before(l.items[tmid].recycleTime) {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return (right + basel + nlen) % nlen
}

func (l *loopQueue) reset() {
	if l.isEmpty() {
		return
	}
Releasing:
	if w := l.detach(); w != nil {
		w.task <- nil
		goto Releasing
	}
	l.items = l.items[:0]
	l.size, l.head, l.tail = 0, 0, 0
}
