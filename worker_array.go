package GoPool

import (
	"errors"
	"time"
)

var (
	errQueueIsFull     = errors.New("the queue is full")
	errQueueIsReleased = errors.New("the queue length is zero")
)

type workerArray interface {
	len() int
	isEmpty() bool
	insert(worker *goWorker) error
	detach() *goWorker
	retrieveExpiry(duration time.Duration) []*goWorker
	reset()
}
type arrayType int

const (
	stackType arrayType = 1 << iota
	loopQueueType
)

func newWorkerArray(aType arrayType, size int) workerArray {
	switch aType {
	case stackType:
		return newWorkerStack(size)
	case loopQueueType:
		return newWorkerLoopQueue(size)
	default:
		return newWorkerStack(size)
	}
}
