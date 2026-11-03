package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", handlePostEvents)
	mux.HandleFunc("GET /campaigns/{campaignID}/stats", handleGetStats)

	addr := ":8080"
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("listening on %s\\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}

	if err := store.Close(); err != nil {
		log.Printf("database close: %v", err)
	}
}

func handlePostEvents(w http.ResponseWriter, r *http.Request) {
	var rawEvents []json.RawMessage

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&rawEvents); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON array"})
		return
	}

	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request must contain one JSON array"})
		return
	}

	if rawEvents == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected a JSON array"})
		return
	}

	accepted, duplicates, conflicts, rejected := 0, 0, 0, 0
	errors := make([]map[string]interface{}, 0)

	for i, raw := range rawEvents {
		var event Event
		if err := json.Unmarshal(raw, &event); err != nil {
			rejected++
			errors = append(errors, map[string]interface{}{"index": i, "error": "invalid event JSON"})
			continue
		}

		status, err := store.Add(event)
		switch status {
		case "accepted":
			accepted++
		case "duplicate":
			duplicates++
		case "conflict":
			conflicts++
			errors = append(errors, map[string]interface{}{"index": i, "error": "event_id already exists with different data"})
		case "invalid":
			rejected++
			errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accepted": accepted, "duplicates": duplicates,
		"conflicts": conflicts, "rejected": rejected, "errors": errors,
	})
}

func handleGetStats(w http.ResponseWriter, r *http.Request) {
	campaignID := r.PathValue("campaignID")
	events := store.GetCampaignEvents(campaignID)

	stats := map[string]interface{}{
		"campaign_id": campaignID,
		"sent":        0, "delivered": 0, "opened": 0, "clicked": 0,
	}

	openedContacts := make(map[string]bool)

	for _, event := range events {
		switch event.Type {
		case "sent":
			stats["sent"] = stats["sent"].(int) + 1
		case "delivered":
			stats["delivered"] = stats["delivered"].(int) + 1
		case "opened":
			openedContacts[event.ContactID] = true
		case "clicked":
			stats["clicked"] = stats["clicked"].(int) + 1
		}
	}

	stats["opened"] = len(openedContacts)
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}
