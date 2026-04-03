package utils

import "sync/atomic"

var (
	TotalRequests  atomic.Uint64
	CacheHits      atomic.Uint64
	ActiveRequests atomic.Int64
	FailedRequests atomic.Uint64
)
