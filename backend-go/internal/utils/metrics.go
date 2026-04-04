package utils

import "sync/atomic"

var (
	TotalRequests  atomic.Uint64
	CacheHits      atomic.Uint64
	ActiveRequests atomic.Int64
	FailedRequests atomic.Uint64

	// Polling speed stats
	TestTasksCompleted atomic.Uint64
	TesterSpeed        atomic.Uint64 // Ends up representing completed tasks per minute
	PendingTestsQueue  atomic.Int64
)
