# Debugging Report

## Bug 1: Duplicate event IDs across batches
- **Problem:** Duplicate IDs were tracked only within each batch.
- **Cause:** The seen map was recreated inside processBatch().
- **Fix:** Moved event ID tracking to a global map shared across batches.
- **Verification:** Output matches expected_output.txt.

## Bug 2: Unique opens counted globally
- **Problem:** The same contact opening in different campaigns counted only once.
- **Cause:** openedBy used contact ID alone as its key.
- **Fix:** Used campaign ID + contact ID as the composite key.
- **Verification:** Output matches expected_output.txt.

## Bug 3: Incorrect daily delivery date
- **Problem:** Daily buckets used local timezone dates.
- **Cause:** Timestamp.Local() converted UTC timestamps to local time.
- **Fix:** Replaced Local() with UTC().
- **Verification:** Daily delivery buckets match expected_output.txt.

## Bug 4: Concurrent counter updates
- **Problem:** Worker goroutines updated shared counters without synchronization.
- **Cause:** Multiple workers accessed the same campaign statistics simultaneously.
- **Fix:** Added a mutex around counter updates in apply().
- **Verification:** go run -race . events.jsonl completed without race warnings.
