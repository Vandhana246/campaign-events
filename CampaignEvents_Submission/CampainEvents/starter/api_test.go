package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostEventsAPI(t *testing.T) {
	oldStore := store
	store = newTestStore(t)
	defer func() { store = oldStore }()

	body := `[
{"event_id":"api-1","campaign_id":"c1","contact_id":"u1","type":"delivered","timestamp":"2026-08-10T09:00:00Z"},
{"event_id":"api-1","campaign_id":"c1","contact_id":"u1","type":"delivered","timestamp":"2026-08-10T09:00:00Z"},
{"event_id":"api-1","campaign_id":"c1","contact_id":"u1","type":"opened","timestamp":"2026-08-10T09:01:00Z"},
{"event_id":"api-2","campaign_id":"c1","contact_id":"u2","type":"invalid","timestamp":"2026-08-10T09:02:00Z"},
{"event_id":"api-3","campaign_id":"c1","contact_id":"u3","type":"clicked","timestamp":"2026-08-10T09:03:00Z"}
]`

	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handlePostEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result struct {
		Accepted   int `json:"accepted"`
		Duplicates int `json:"duplicates"`
		Conflicts  int `json:"conflicts"`
		Rejected   int `json:"rejected"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result.Accepted != 2 || result.Duplicates != 1 ||
		result.Conflicts != 1 || result.Rejected != 1 {
		t.Fatalf("unexpected API summary: %+v", result)
	}
}

func TestCampaignStatsAPI(t *testing.T) {
	oldStore := store
	store = newTestStore(t)
	defer func() { store = oldStore }()

	events := []Event{
		{"s1", "c1", "u1", "sent", "2026-08-10T09:00:00Z", nil},
		{"d1", "c1", "u1", "delivered", "2026-08-10T09:01:00Z", nil},
		{"o1", "c1", "u1", "opened", "2026-08-10T09:02:00Z", nil},
		{"o2", "c1", "u1", "opened", "2026-08-10T09:03:00Z", nil},
		{"o3", "c1", "u2", "opened", "2026-08-10T09:04:00Z", nil},
		{"c1", "c1", "u1", "clicked", "2026-08-10T09:05:00Z", nil},
	}

	for _, event := range events {
		if status, err := store.Add(event); err != nil || status != "accepted" {
			t.Fatalf("failed to insert test event: %s, %v", status, err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/campaigns/c1/stats", nil)
	req.SetPathValue("campaignID", "c1")
	rec := httptest.NewRecorder()

	handleGetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	var result struct {
		Sent      int `json:"sent"`
		Delivered int `json:"delivered"`
		Opened    int `json:"opened"`
		Clicked   int `json:"clicked"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result.Sent != 1 || result.Delivered != 1 ||
		result.Opened != 2 || result.Clicked != 1 {
		t.Fatalf("unexpected statistics: %+v", result)
	}
}

func TestEmptyCampaignStatsAPI(t *testing.T) {
	oldStore := store
	store = newTestStore(t)
	defer func() { store = oldStore }()

	req := httptest.NewRequest(http.MethodGet, "/campaigns/missing/stats", nil)
	req.SetPathValue("campaignID", "missing")
	rec := httptest.NewRecorder()

	handleGetStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"sent", "delivered", "opened", "clicked"} {
		if result[key] != float64(0) {
			t.Errorf("expected %s to be zero, got %v", key, result[key])
		}
	}
}
