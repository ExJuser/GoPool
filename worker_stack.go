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
	expiryTime := time.Now().Add(-duration)
	w.expiry = w.expiry[:0]
	if index := w.binarySearch(0, n-1, expiryTime); index != -1 {
		w.expiry = append(w.expiry, w.items[:index+1]...)
		m := copy(w.items, w.items[index+1:])
		for i := m; i < n; i++ {
			w.items[i] = nil
		}
		w.items = w.items[m:]
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
