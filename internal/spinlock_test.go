package internal

import (
	"sync"
	"testing"
)

func BenchmarkMutex(b *testing.B) {
	m := sync.Mutex{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Lock()
			//nolint:staticcheck
			m.Unlock()
		}
	})
}

func BenchmarkSpinLockOrigin(b *testing.B) {
	lock := NewSpinLockOrigin()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lock.Lock()
			//nolint:staticcheck
			lock.Unlock()
		}
	})
}

func BenchmarkSpinLockBackoff(b *testing.B) {
	lock := NewSpinLockBackoff()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lock.Lock()
			//nolint:staticcheck
			lock.Unlock()
		}
	})
}
