package GoPool

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewLoopQueue(t *testing.T) {
	size := 100
	q := newWorkerLoopQueue(size)
	assert.EqualValues(t, 0, q.len(), "Len error")
	assert.Equal(t, true, q.isEmpty(), "IsEmpty error")
	assert.Nil(t, q.detach(), "Detach error")
}
