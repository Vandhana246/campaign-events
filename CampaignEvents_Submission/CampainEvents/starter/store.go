package main

import (
	"database/sql"
	"encoding/json"
	"reflect"

	_ "github.com/mattn/go-sqlite3"
)

type EventStore struct {
	db *sql.DB
}

func NewEventStore() *EventStore {
	store, err := OpenEventStore("campaign_events.db")
	if err != nil {
		panic(err)
	}
	return store
}

func OpenEventStore(path string) (*EventStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	_, err = db.Exec(`
PRAGMA journal_mode=WAL;

CREATE TABLE IF NOT EXISTS events (
event_id TEXT PRIMARY KEY,
campaign_id TEXT NOT NULL,
contact_id TEXT NOT NULL,
type TEXT NOT NULL,
timestamp TEXT NOT NULL,
metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_campaign
ON events(campaign_id);
`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &EventStore{db: db}, nil
}

func (s *EventStore) Close() error {
	return s.db.Close()
}

func (s *EventStore) Add(event Event) (string, error) {
	if err := validateEvent(&event); err != nil {
		return "invalid", err
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return "invalid", err
	}

	result, err := s.db.Exec(`
INSERT INTO events
(event_id, campaign_id, contact_id, type, timestamp, metadata)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(event_id) DO NOTHING
`, event.EventID, event.CampaignID, event.ContactID,
		event.Type, event.Timestamp, string(metadata))

	if err != nil {
		return "invalid", err
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		return "invalid", err
	}

	if inserted == 1 {
		return "accepted", nil
	}

	existing, err := s.getEvent(event.EventID)
	if err != nil {
		return "invalid", err
	}

	if reflect.DeepEqual(existing, event) {
		return "duplicate", nil
	}

	return "conflict", nil
}

func (s *EventStore) getEvent(eventID string) (Event, error) {
	var event Event
	var metadata string

	err := s.db.QueryRow(`
SELECT event_id, campaign_id, contact_id, type, timestamp, metadata
FROM events WHERE event_id = ?
`, eventID).Scan(
		&event.EventID, &event.CampaignID, &event.ContactID,
		&event.Type, &event.Timestamp, &metadata,
	)
	if err != nil {
		return Event{}, err
	}

	if metadata != "" && metadata != "null" {
		if err := json.Unmarshal([]byte(metadata), &event.Metadata); err != nil {
			return Event{}, err
		}
	}

	return event, nil
}

func (s *EventStore) GetCampaignEvents(campaignID string) []Event {
	rows, err := s.db.Query(`
SELECT event_id, campaign_id, contact_id, type, timestamp, metadata
FROM events WHERE campaign_id = ?
ORDER BY timestamp, event_id
`, campaignID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var events []Event

	for rows.Next() {
		var event Event
		var metadata string

		if err := rows.Scan(
			&event.EventID, &event.CampaignID, &event.ContactID,
			&event.Type, &event.Timestamp, &metadata,
		); err != nil {
			return nil
		}

		if metadata != "" && metadata != "null" {
			if err := json.Unmarshal([]byte(metadata), &event.Metadata); err != nil {
				return nil
			}
		}

		events = append(events, event)
	}

	if rows.Err() != nil {
		return nil
	}

	return events
}

var store = NewEventStore()
