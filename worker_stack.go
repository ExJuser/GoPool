package GoPool

import "time"

type workerStack struct {
	items  []*goWorker
	expiry []*goWorker
	size   int
}

func newWorkerStack(size int) *workerStack {
	return &workerStack{
		items: make([]*goWorker, 0, size),
		size:  size,
	}
}

func (w *workerStack) len() int {
	return len(w.items)
}

func (w *workerStack) isEmpty() bool {
	return len(w.items) == 0
}

func (w *workerStack) insert(worker *goWorker) error {
	w.items = append(w.items, worker)
	return nil
}

func (w *workerStack) detach() *goWorker {
	l := w.len()
	if l == 0 {
		return nil
	}
	//栈 后进先出
	worker := w.items[l-1]
	w.items[l-1] = nil
	w.items = w.items[:l-1]
	return worker
}

func (w *workerStack) retrieveExpiry(duration time.Duration) []*goWorker {
	n := w.len()
	if n == 0 {
		return nil
	}
	//已经有 duration 没有被使用过的worker认为是已经过期
	//即recycleTime早于expiryTime的worker被认为是过期
	expiryTime := time.Now().Add(-duration)
	w.expiry = w.expiry[:0]
	//由于先加入栈的一定更早过期 采用二分查找快速找到过期的worker
	if index := w.binarySearch(0, n-1, expiryTime); index != -1 {
		w.expiry = append(w.expiry, w.items[:index+1]...)
		//将没有过期的worker填充到items的前面
		m := copy(w.items, w.items[index+1:])
		//清空已经过期的worker的内存防止内存泄漏
		for i := m; i < n; i++ {
			w.items[i] = nil
		}
		w.items = w.items[:m]
	}
	return w.expiry
}
func (w *workerStack) binarySearch(left, right int, expiryTime time.Time) int {
	var mid int
	for left <= right {
		mid = (right-left)/2 + left
		if expiryTime.Before(w.items[mid].recycleTime) {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return right
}
func (w *workerStack) reset() {
	for i := 0; i < w.len(); i++ {
		w.items[i].task <- nil
		w.items[i] = nil
	}
	w.items = w.items[:0]
}
