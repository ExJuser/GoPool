//go:build !windows
// +build !windows

package GoPool

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewLoopQueue(t *testing.T) {
	size := 100
	q := newWorkerLoopQueue(size)
	assert.EqualValues(t, 0, q.len(), "Len error")
	assert.Equal(t, true, q.isEmpty(), "IsEmpty error")
	assert.Nil(t, q.detach(), "Detach error")
}

func TestLoopQueue(t *testing.T) {
	size := 10
	q := newWorkerLoopQueue(size)

	for i := 0; i < 5; i++ {
		err := q.insert(&goWorker{
			recycleTime: time.Now(),
		})
		if err != nil {
			break
		}
	}
	assert.EqualValues(t, 5, q.len(), "Len error")
	v := q.detach()
	t.Log(v)
	assert.EqualValues(t, 4, q.len(), "Len error")

	time.Sleep(time.Second)

	for i := 0; i < 6; i++ {
		err := q.insert(&goWorker{recycleTime: time.Now()})
		if err != nil {
			break
		}
	}
	assert.EqualValues(t, 10, q.len(), "Len error")
	err := q.insert(&goWorker{recycleTime: time.Now()})
	assert.Error(t, err, "errQueueIsFull")

	q.retrieveExpiry(time.Second)
	//time.Sleep(time.Second)之前的4个worker过期
	assert.EqualValues(t, 6, q.len(), "Len error")
}

func TestSearch(t *testing.T) {
	size := 10
	q := newWorkerLoopQueue(size)

	expiry1 := time.Now()

	_ = q.insert(&goWorker{recycleTime: time.Now()})
	assert.EqualValues(t, 0, q.binarySearch(time.Now()), "index should be 0")
	//没有worker在expiry之前被加进队列
	assert.EqualValues(t, -1, q.binarySearch(expiry1), "index should be -1")

	expiry2 := time.Now()
	_ = q.insert(&goWorker{recycleTime: time.Now()})
	//没有worker在expiry之前被加进队列
	assert.EqualValues(t, -1, q.binarySearch(expiry1), "index should be -1")

	assert.EqualValues(t, 0, q.binarySearch(expiry2), "index should be 0")
	//两个加入的worker都在time.Now()之前
	assert.EqualValues(t, 1, q.binarySearch(time.Now()), "index should be 1")

	//q.Len()==7
	for i := 0; i < 5; i++ {
		_ = q.insert(&goWorker{recycleTime: time.Now()})
	}

	expiry3 := time.Now()
	_ = q.insert(&goWorker{recycleTime: expiry3})

	//q.Len()==10
	var err error
	for !errors.Is(err, errQueueIsFull) {
		err = q.insert(&goWorker{recycleTime: time.Now()})
	}

	assert.EqualValues(t, 7, q.binarySearch(expiry3), "index should be 7")

	for i := 0; i < 6; i++ {
		_ = q.detach()
	}

	expiry4 := time.Now()
	_ = q.insert(&goWorker{recycleTime: time.Now()})

	assert.EqualValues(t, 0, q.binarySearch(expiry4), "index should be 0")

	for i := 0; i < 3; i++ {
		_ = q.detach()
	}
	expiry5 := time.Now()
	_ = q.insert(&goWorker{recycleTime: expiry5})

	assert.EqualValues(t, 5, q.binarySearch(expiry5), "index should be 5")

	for i := 0; i < 3; i++ {
		_ = q.insert(&goWorker{recycleTime: time.Now()})
	}

	assert.EqualValues(t, -1, q.binarySearch(expiry2), "index should be -1")

	assert.EqualValues(t, 9, q.binarySearch(q.items[9].recycleTime), "index should be 9")
	assert.EqualValues(t, 8, q.binarySearch(time.Now()), "index should be 8")
}
