package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func newTestStore(t *testing.T) *EventStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := OpenEventStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	return s
}

func TestStoreAddAndDuplicates(t *testing.T) {
	s := newTestStore(t)

	e := Event{
		EventID: "e1", CampaignID: "c1",
		ContactID: "u1", Type: "delivered",
		Timestamp: "2026-08-10T09:00:00+05:30",
	}

	status, err := s.Add(e)
	if err != nil || status != "accepted" {
		t.Fatalf("expected accepted, got %s, %v", status, err)
	}

	stored := s.GetCampaignEvents("c1")
	if len(stored) != 1 || stored[0].Timestamp != "2026-08-10T03:30:00Z" {
		t.Fatalf("timestamp not normalized: %+v", stored)
	}

	status, err = s.Add(e)
	if err != nil || status != "duplicate" {
		t.Fatalf("expected duplicate, got %s, %v", status, err)
	}

	e.Type = "opened"
	status, err = s.Add(e)
	if err != nil || status != "conflict" {
		t.Fatalf("expected conflict, got %s, %v", status, err)
	}
}

func TestStoreRejectsInvalidEvent(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Add(Event{EventID: "e1"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestStorePersistsAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persistent.db")

	s, err := OpenEventStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	e := Event{
		EventID:    "persist-1",
		CampaignID: "campaign-1",
		ContactID:  "contact-1",
		Type:       "delivered",
		Timestamp:  "2026-08-10T09:00:00Z",
	}

	status, err := s.Add(e)
	if err != nil || status != "accepted" {
		t.Fatalf("first insert: %s, %v", status, err)
	}
	s.Close()

	s, err = OpenEventStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	status, err = s.Add(e)
	if err != nil || status != "duplicate" {
		t.Fatalf("expected persisted duplicate, got %s, %v", status, err)
	}

	events := s.GetCampaignEvents("campaign-1")
	if len(events) != 1 {
		t.Fatalf("expected 1 persisted event, got %d", len(events))
	}
}

func TestConcurrentDuplicateInsertion(t *testing.T) {
	s := newTestStore(t)

	e := Event{
		EventID:    "concurrent-1",
		CampaignID: "campaign-1",
		ContactID:  "contact-1",
		Type:       "delivered",
		Timestamp:  "2026-08-10T09:00:00Z",
	}

	const workers = 20

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted, duplicates := 0, 0
	errs := make([]error, 0)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			status, err := s.Add(e)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, err)
				return
			}

			switch status {
			case "accepted":
				accepted++
			case "duplicate":
				duplicates++
			default:
				errs = append(errs, fmt.Errorf("unexpected status: %s", status))
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("concurrent insertion errors: %v", errs)
	}
	if accepted != 1 || duplicates != workers-1 {
		t.Fatalf("expected 1 accepted and 19 duplicates; got %d accepted, %d duplicates", accepted, duplicates)
	}
}

func TestOutOfOrderEventsAccepted(t *testing.T) {
	s := newTestStore(t)

	events := []Event{
		{
			EventID:    "late-open",
			CampaignID: "campaign-order",
			ContactID:  "contact-1",
			Type:       "opened",
			Timestamp:  "2026-08-10T10:00:00Z",
		},
		{
			EventID:    "earlier-delivery",
			CampaignID: "campaign-order",
			ContactID:  "contact-1",
			Type:       "delivered",
			Timestamp:  "2026-08-10T09:00:00Z",
		},
	}

	for _, event := range events {
		status, err := s.Add(event)
		if err != nil || status != "accepted" {
			t.Fatalf("expected out-of-order event to be accepted, got %s, %v", status, err)
		}
	}

	if got := len(s.GetCampaignEvents("campaign-order")); got != 2 {
		t.Fatalf("expected 2 stored events, got %d", got)
	}
}
