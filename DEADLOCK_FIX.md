# PostgreSQL Deadlock Fix

## Problem
The application was experiencing PostgreSQL deadlocks when testing multiple Ollama endpoints concurrently. The logs showed:

```
[tester] could not upsert ai_model deepseek-r1:1.5b: pq: deadlock detected
[tester] commit error: pq: Could not complete operation in a failed transaction
```

### Root Cause
Multiple concurrent goroutines executing `INSERT ... ON CONFLICT` operations on the `ai_models` table caused circular wait conditions:

1. Transaction A tries to insert model X, acquires an exclusive lock on the index entry
2. Transaction B tries to insert the same model X, waits for the lock
3. Transaction A then tries to insert model Y which Transaction B is currently working on
4. **Deadlock**: circular dependency between transactions

This is a well-known issue with PostgreSQL's `INSERT ... ON CONFLICT` when multiple concurrent transactions attempt to upsert the same rows.

## Solution
Implemented PostgreSQL advisory locks to serialize access to model insertions on a per-model basis.

### How It Works
1. **Hash-based Lock IDs**: Created `getModelLockID()` function that generates a deterministic int64 lock ID from the model name+tag combination using FNV-1a hash
2. **Advisory Locks**: Before each upsert, acquire `pg_advisory_xact_lock(lockID)` which:
   - Blocks until the lock is available for that specific model
   - Ensures only one transaction can insert/update a specific model at a time
   - Automatically releases when the transaction commits or rolls back
   - Different models can still be processed concurrently (different lock IDs)

### Code Changes
**File**: `backend-go/internal/services/ollama.go`

1. Added `hash/fnv` import
2. Added `getModelLockID(name, tag string) int64` helper function
3. Modified `executeTask()` to acquire advisory lock before each model upsert

```go
// Before upsert, acquire lock for this specific model
lockID := getModelLockID(mr.ModelName, mr.ModelTag)
_, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", lockID)
if err != nil {
    log.Printf("[tester] could not acquire advisory lock for %s:%s: %v", mr.ModelName, mr.ModelTag, err)
    continue
}

// Now safely upsert the model
var modelID int
err = tx.QueryRow(`
    INSERT INTO ai_models (name, tag) VALUES ($1, $2)
    ON CONFLICT (name, tag) DO UPDATE SET name = EXCLUDED.name
    RETURNING id`,
    mr.ModelName, mr.ModelTag,
).Scan(&modelID)
```

## Testing
- Added `TestGetModelLockID()` to verify:
  - Deterministic output (same input always produces same lock ID)
  - Different models produce different lock IDs
  - Lock IDs are always positive (valid int64 range)
- All existing tests pass
- Full test suite runs successfully

## Benefits
✅ Eliminates deadlocks on `ai_models` table  
✅ Maintains concurrent processing of different models  
✅ Transaction-scoped locks (auto-cleanup)  
✅ No schema changes required  
✅ Minimal performance impact (lock acquisition is fast)  

## Alternative Approaches Considered
1. **SERIALIZABLE isolation level**: Too restrictive, would reduce concurrency significantly
2. **Table-level locking**: Would prevent all concurrent model inserts, defeating the purpose of parallelism
3. **Application-level mutex**: Wouldn't work across multiple application instances
4. **Retry logic**: Would mask the problem but not solve it, wasting resources on retries

The advisory lock approach provides the best balance of correctness and performance.
