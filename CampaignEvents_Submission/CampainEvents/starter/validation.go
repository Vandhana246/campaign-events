package main

import (
	"fmt"
	"strings"
	"time"
)

var validEventTypes = map[string]bool{
	"sent":      true,
	"delivered": true,
	"opened":    true,
	"clicked":   true,
}

func validateEvent(e *Event) error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(e.CampaignID) == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if strings.TrimSpace(e.ContactID) == "" {
		return fmt.Errorf("contact_id is required")
	}
	if !validEventTypes[e.Type] {
		return fmt.Errorf("invalid event type: %s", e.Type)
	}

	t, err := time.Parse(time.RFC3339, e.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %s", e.Timestamp)
	}

	e.Timestamp = t.UTC().Format(time.RFC3339Nano)
	return nil
}
