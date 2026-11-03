# Scaling to 100 Million Events per Day

100 million events/day averages approximately 1,157 events/second. Actual peak traffic will be higher and must be measured.

## Architecture
1. Stateless webhook API instances behind a load balancer.
2. Validate and authenticate incoming requests.
3. Publish accepted events to a replicated Kafka-style durable queue.
4. Use a durable idempotency key, preferably (provider_id, event_id).
5. Partition events by campaign or a stable campaign hash.
6. Use stream processors to calculate campaign statistics.
7. Store raw events in partitioned time-based storage.
8. Maintain precomputed campaign aggregates for fast queries.

## Reliability
- Acknowledge only after durable acceptance.
- Use idempotent consumers to handle retries.
- Handle late events using event timestamps and reconciliation.
- Use UTC for daily aggregation.
- Support replay and backfills.
- Monitor queue lag, failures, duplicates and processing latency.

## Trade-offs
Exactly-once processing is expensive. Prefer at-least-once delivery with idempotent writes. Unique opens require state keyed by campaign and contact. Load-test peak traffic and recovery before production deployment.
