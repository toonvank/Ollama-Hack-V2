package utils

import (
	"sync"
	"testing"
)

func TestMetricsConcurrency(t *testing.T) {
	// Reset counters
	TotalRequests.Store(0)
	CacheHits.Store(0)
	ActiveRequests.Store(0)
	FailedRequests.Store(0)
	TestTasksCompleted.Store(0)
	TesterSpeed.Store(0)
	PendingTestsQueue.Store(0)

	var wg sync.WaitGroup
	iterations := 1000

	// Test TotalRequests concurrent increments
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			TotalRequests.Add(1)
		}()
	}
	wg.Wait()

	if TotalRequests.Load() != uint64(iterations) {
		t.Errorf("TotalRequests: expected %d, got %d", iterations, TotalRequests.Load())
	}

	// Test ActiveRequests concurrent increments/decrements
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ActiveRequests.Add(1)
		}()
		go func() {
			defer wg.Done()
			ActiveRequests.Add(-1)
		}()
	}
	wg.Wait()

	if ActiveRequests.Load() != 0 {
		t.Errorf("ActiveRequests: expected 0, got %d", ActiveRequests.Load())
	}

	// Test CacheHits
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CacheHits.Add(1)
		}()
	}
	wg.Wait()

	if CacheHits.Load() != uint64(iterations) {
		t.Errorf("CacheHits: expected %d, got %d", iterations, CacheHits.Load())
	}

	// Test FailedRequests
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			FailedRequests.Add(1)
		}()
	}
	wg.Wait()

	if FailedRequests.Load() != uint64(iterations) {
		t.Errorf("FailedRequests: expected %d, got %d", iterations, FailedRequests.Load())
	}

	// Test TestTasksCompleted
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			TestTasksCompleted.Add(1)
		}()
	}
	wg.Wait()

	if TestTasksCompleted.Load() != uint64(iterations) {
		t.Errorf("TestTasksCompleted: expected %d, got %d", iterations, TestTasksCompleted.Load())
	}

	// Test PendingTestsQueue
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			PendingTestsQueue.Add(1)
		}()
		go func() {
			defer wg.Done()
			PendingTestsQueue.Add(-1)
		}()
	}
	wg.Wait()

	if PendingTestsQueue.Load() != 0 {
		t.Errorf("PendingTestsQueue: expected 0, got %d", PendingTestsQueue.Load())
	}
}

func TestMetricsInitialValues(t *testing.T) {
	// Verify atomic counters can be read without issues
	_ = TotalRequests.Load()
	_ = CacheHits.Load()
	_ = ActiveRequests.Load()
	_ = FailedRequests.Load()
	_ = TestTasksCompleted.Load()
	_ = TesterSpeed.Load()
	_ = PendingTestsQueue.Load()
}
