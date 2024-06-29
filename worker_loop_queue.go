package GoPool

import (
	"time"
)

type loopQueue struct {
	items            []*goWorker
	expiry           []*goWorker
	head, tail, size int  //队列的头索引、尾索引和大小
	isFull           bool // 队列是否已满的标志
}

func newWorkerLoopQueue(size int) *loopQueue {
	return &loopQueue{
		items: make([]*goWorker, size), //循环队列 分配指定大小的存储空间
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
	// 将Worker插入尾部 并移动末尾
	l.items[l.tail] = worker
	l.tail++
	if l.tail == l.size { //如果尾部到达队列末端重置到起始位置
		l.tail = 0
	}
	// 如果插入后头尾相遇代表队列已满
	if l.tail == l.head {
		l.isFull = true
	}
	return nil
}

func (l *loopQueue) detach() *goWorker {
	if l.isEmpty() {
		return nil
	}
	//去除队列头部的worker 并移动头部
	w := l.items[l.head]
	l.items[l.head] = nil
	l.head++
	if l.head == l.size { //如果头部到达队列末端重置到起始位置
		l.head = 0
	}
	l.isFull = false // 移除后队列不满
	return w
}

// 检索并返回已经过期的worker
func (l *loopQueue) retrieveExpiry(duration time.Duration) []*goWorker {
	expiryTime := time.Now().Add(-duration)
	index := l.binarySearch(expiryTime)
	if index == -1 {
		return nil
	}
	l.expiry = l.expiry[:0] // 清空过期列表
	if l.head <= index {    //从head到index都是过期的worker
		l.expiry = append(l.expiry, l.items[l.head:index+1]...)
		for i := l.head; i < index+1; i++ {
			l.items[i] = nil
		}
	} else { //从head到队列末尾 再循环到index都是过期的worker
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
		l.isFull = false // 有过期Worker时队列不满
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
	//Releasing:
	//	if w := l.detach(); w != nil {
	//		w.task <- nil
	//		goto Releasing
	//	}
	for {
		w := l.detach()
		if w == nil {
			break
		}
		w.task = nil
	}
	//清空队列
	l.items = l.items[:0]
	l.size, l.head, l.tail = 0, 0, 0
}
