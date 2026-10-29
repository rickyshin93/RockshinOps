package metrics

import (
	"testing"
	"time"
)

func BenchmarkNativeCollectors(b *testing.B) {
	b.Run("hardware", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			readNative(2 * time.Second)
		}
	})
	b.Run("processes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			readProcesses()
		}
	})
}
