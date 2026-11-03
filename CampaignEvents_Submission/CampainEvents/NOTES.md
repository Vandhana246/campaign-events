# Campaign Events – Engineering Notes

## 1. Problem Interpretation
Receive marketing webhook events, validate and store them, prevent duplicate processing, handle late and out-of-order events, and calculate campaign statistics.

## 2. Assumptions and Business Rules
- event_id is globally unique.
- The first valid event is retained.
- Identical repeats are duplicates and are not inserted again.
- Reuse of an event ID with different data is a conflict.
- Sent, delivered and clicked count unique accepted events.
- Opened counts distinct contacts per campaign.
- Timestamps must be RFC3339 and are normalized to UTC.
- Late and out-of-order events are accepted.
- Invalid records are rejected individually; valid records in the same batch are processed.
- Metadata is optional.

## 3. Ambiguities and Product Manager Questions
1. Are event IDs globally unique or provider-specific?
2. Does opened mean total opens or unique contacts?
3. Should clicked count unique clicks or unique contacts?
4. How should conflicting event IDs be resolved?
5. What is the event retention period?
6. Should partial batches be accepted?
7. Are provider-specific event types supported?
8. Should statistics be recalculated or incrementally maintained?

## 4. Implementation Priorities
- P0: Validation, idempotency, accurate statistics and concurrency safety.
- P1: Persistent storage, automated tests and error reporting.
- P2: Pagination, monitoring and dashboards.

## 5. Completed Implementation
- Go HTTP service using net/http.
- POST /events with partial batch processing.
- GET /campaigns/{campaign_id}/stats.
- SQLite persistence through database/sql.
- event_id primary key for database-level uniqueness.
- Duplicate and conflicting payload detection.
- UTC timestamp normalization.
- Mutex-protected test concurrency and database uniqueness.
- Automated store, API, persistence and concurrency tests.

## 6. Verification Results
- Valid event insertion and duplicate handling verified.
- Conflicting IDs and invalid events verified.
- Campaign statistics verified.
- Seed data: 210 accepted, 15 duplicates, 3 conflicts and 7 rejected on first ingestion.
- Re-ingestion: 0 accepted, 225 duplicates, 3 conflicts and 7 rejected.
- Statistics remained unchanged after re-ingestion.
- Database persistence across server restart verified.
- Go race detector passed for the test suites and debugging program.
- Debugging output matched expected_output.txt.

## 7. Scale Memo – 100 Million Events per Day
100 million events per day averages approximately 1,157 events per second. Peak traffic will be higher.

Proposed architecture:
1. Stateless API instances behind a load balancer.
2. Request validation and provider authentication.
3. Replicated durable message queue such as Kafka.
4. Idempotency key using provider_id and event_id.
5. Partitioning by campaign or stable campaign hash.
6. Stream processors for campaign statistics.
7. Partitioned time-based raw event storage.
8. Precomputed aggregates for fast statistics queries.

Reliability:
- Acknowledge only after durable acceptance.
- Use idempotent consumers.
- Handle late events and reconciliation.
- Use UTC for daily aggregation.
- Support replay and backfills.
- Monitor queue lag, failures, duplicates and latency.

Trade-offs:
Prefer at-least-once delivery with idempotent writes over expensive exactly-once processing. Maintain unique-open state by campaign and contact. Load-test peak traffic and recovery.

## 8. Angry Marketer Investigation
Scenario: delivered increased but opened decreased.

Possible explanations:
- More contacts received the campaign but a smaller proportion opened it.
- Opens arrived late.
- Email privacy features or blocked images affected tracking.
- Provider integrations or tracking pixels changed.
- Duplicate, missing or delayed events affected reporting.
- Audience, subject line or send time changed.

Investigation:
1. Confirm campaign and reporting time window.
2. Compare unique delivered and opened contacts.
3. Calculate open rate using a consistent denominator.
4. Segment by provider, domain, audience and send time.
5. Check ingestion delays, duplicates and invalid records.
6. Verify tracking pixels and provider reports.
7. Compare after a consistent observation window.

Response:
"We are checking whether the difference reflects genuine engagement or a tracking/reporting issue. We will compare unique-contact metrics, inspect provider-level data and event delays, and verify open tracking before recommending campaign changes."

## 9. Limitations and Skipped Work
- SQLite is suitable for this assignment but is not a distributed high-volume database.
- No authentication or rate limiting.
- No durable message queue.
- No optional paginated event-list endpoint.
- No frontend or deployment.
- Campaign statistics are calculated from stored events rather than precomputed aggregates.
- Production backup, retention and monitoring policies are not implemented.

## 10. If I Had Another Day
- Add more HTTP status-code and malformed-request tests.
- Add paginated campaign event listing.
- Add database backup and retention policies.
- Add structured logging and operational metrics.
- Add load tests for concurrent ingestion.
- Review SQLite transaction and connection settings under realistic load.
- Improve API error documentation.
