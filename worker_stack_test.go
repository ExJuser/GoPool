package GoPool

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewWorkerStack(t *testing.T) {
	size := 100
	q := newWorkerStack(size)
	assert.EqualValues(t, 0, q.len(), "Len error")
	assert.Equal(t, true, q.isEmpty(), "IsEmpty error")
	assert.Nil(t, q.detach(), "Dequeue error")
}

func TestWorkerStack(t *testing.T) {
	q := newWorkerArray(arrayType(-1), 0)
	for i := 0; i < 5; i++ {
		err := q.insert(&goWorker{recycleTime: time.Now()})
		if err != nil {
			break
		}
	}
	assert.EqualValues(t, 5, q.len(), "Len error")
	expired := time.Now()
	err := q.insert(&goWorker{recycleTime: expired})
	if err != nil {
		t.Fatal("enqueue error")
	}
	time.Sleep(time.Second)
	for i := 0; i < 6; i++ {
		err := q.insert(&goWorker{
			recycleTime: time.Now(),
		})
		if err != nil {
			t.Fatal("enqueue error")
		}
	}
	assert.EqualValues(t, 12, q.len(), "Len error")
	q.retrieveExpiry(time.Second)
	assert.EqualValues(t, 6, q.len(), "Len error")
}
