# Campaign Events API

A Go HTTP service for ingesting, validating, and storing marketing campaign events using SQLite.

## Requirements
- Go 1.22+
- GCC for the SQLite driver

## Run
    go mod tidy
    go run .

## Endpoints
- POST /events: accepts a JSON array and reports accepted, duplicate, conflicting, and rejected events.
- GET /campaigns/{campaign_id}/stats: returns sent, delivered, opened, and clicked counts.

## Behaviour
- Validates required fields, event types, and RFC3339 timestamps.
- Normalizes timestamps to UTC.
- Uses globally unique event IDs.
- Identical repeats are duplicates; different payloads with the same ID are conflicts.
- Accepts late and out-of-order events.
- Persists events in SQLite across restarts.

## Tests
    go build ./...
    go vet ./...
    go test ./...
    go test -race ./...
