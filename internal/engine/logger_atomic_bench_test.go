// SPDX-License-Identifier: MIT
// Copyright (C) 2026 0verkilll

package engine

import "testing"

// BenchmarkGetLogger measures the cost of the lock-free atomic read path
// used by engine.GetLogger.
func BenchmarkGetLogger(b *testing.B) {
	SetLogger(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetLogger()
	}
}
