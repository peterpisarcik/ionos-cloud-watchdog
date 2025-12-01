package feed

import (
	"testing"
	"time"
)

func TestAnalyzeEntries_NoActiveIncidents(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	entries := []Entry{
		{
			Title:   "All systems operational",
			Updated: now,
			Content: "No ongoing work",
		},
	}

	result := analyzeEntries(entries)

	if result.Status != StatusOK {
		t.Fatalf("expected status %s, got %s", StatusOK, result.Status)
	}
	if len(result.ActiveIncidents) != 0 {
		t.Fatalf("expected no active incidents, got %d", len(result.ActiveIncidents))
	}
	if result.Message != "No active incidents" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestAnalyzeEntries_SingleActiveIncident(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	entry := Entry{
		Title:   "Network latency in FRA",
		Updated: now,
		Content: "We are investigating connectivity issues",
	}

	result := analyzeEntries([]Entry{entry})

	if result.Status != StatusWarning {
		t.Fatalf("expected status %s, got %s", StatusWarning, result.Status)
	}
	if len(result.ActiveIncidents) != 1 {
		t.Fatalf("expected 1 active incident, got %d", len(result.ActiveIncidents))
	}
	expectedMsg := "1 active incident: " + entry.Title
	if result.Message != expectedMsg {
		t.Fatalf("expected message %q, got %q", expectedMsg, result.Message)
	}
}

func TestAnalyzeEntries_MultipleActiveIncidents(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	entries := []Entry{
		{
			Title:   "Issue one",
			Updated: now,
			Content: "Investigating a problem",
		},
		{
			Title:   "Issue two",
			Updated: now,
			Content: "Currently impacting customers",
		},
	}

	result := analyzeEntries(entries)

	if result.Status != StatusCritical {
		t.Fatalf("expected status %s, got %s", StatusCritical, result.Status)
	}
	if len(result.ActiveIncidents) != 2 {
		t.Fatalf("expected 2 active incidents, got %d", len(result.ActiveIncidents))
	}
	if result.Message != "2 active incidents" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestAnalyzeEntries_IgnoresResolved(t *testing.T) {
	now := time.Now().Format(time.RFC3339)
	entry := Entry{
		Title:   "Earlier outage",
		Updated: now,
		Content: "Issue resolved for all customers",
	}

	result := analyzeEntries([]Entry{entry})

	if result.Status != StatusOK {
		t.Fatalf("expected status %s, got %s", StatusOK, result.Status)
	}
	if len(result.ActiveIncidents) != 0 {
		t.Fatalf("expected resolved incident to be ignored")
	}
}
