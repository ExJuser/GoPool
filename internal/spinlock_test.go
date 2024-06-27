package internal

import (
	"sync"
	"testing"
)

/**
benchmark results
goos: windows
goarch: amd64
pkg: GoPool/internal
cpu: 13th Gen Intel(R) Core(TM) i5-13600K
BenchmarkMutex-20       	    20655360                56.97 ns/op
BenchmarkSpinLockOrigin-20      86752840                12.83 ns/op
BenchmarkSpinLockBackoff-20     95682086                12.65 ns/op
*/

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
