package GoPool

import (
	"context"
	"sync"
)

type Pool struct {
	capacity      int32
	running       int32
	lock          sync.Locker
	workers       workerArray
	state         int32
	cond          *sync.Cond
	workerCache   sync.Pool
	waiting       int32
	heartbeatDone int32
	stopHeartBeat context.CancelFunc
	options       *Options
}

func NewPool(size int, options ...Option) (*Pool, error) {
	loadoptions
}
