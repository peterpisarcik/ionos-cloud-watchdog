package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	Title   string `xml:"title"`
	Link    Link   `xml:"link"`
	Updated string `xml:"updated"`
	Content string `xml:"content"`
}

type Link struct {
	Href string `xml:"href,attr"`
}

type Status string

const (
	StatusOK       Status = "OK"
	StatusWarning  Status = "WARNING"
	StatusCritical Status = "CRITICAL"
)

type StatusResult struct {
	Status         Status
	ActiveIncidents []Entry
	Message        string
}

var activeKeywords = []string{
	"investigating",
	"identified",
	"monitoring",
	"in progress",
	"currently",
}

var resolvedKeywords = []string{
	"resolved",
	"completed",
	"no customer impact",
}

func CheckStatus() (*StatusResult, error) {
	resp, err := http.Get("https://status.ionos.cloud/history.atom")
	if err != nil {
		return nil, fmt.Errorf("error fetching status page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	var feed Feed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, fmt.Errorf("error parsing atom feed: %w", err)
	}

	return analyzeEntries(feed.Entries), nil
}

func analyzeEntries(entries []Entry) *StatusResult {
	var activeIncidents []Entry
	cutoff := time.Now().Add(-24 * time.Hour)

	for _, entry := range entries {
		updated, err := time.Parse(time.RFC3339, entry.Updated)
		if err != nil {
			continue
		}

		// Only check recent entries (last 24 hours)
		if updated.Before(cutoff) {
			continue
		}

		contentLower := strings.ToLower(entry.Content)

		// Check if resolved
		isResolved := false
		for _, keyword := range resolvedKeywords {
			if strings.Contains(contentLower, keyword) {
				isResolved = true
				break
			}
		}

		if isResolved {
			continue
		}

		// Check if active
		for _, keyword := range activeKeywords {
			if strings.Contains(contentLower, keyword) {
				activeIncidents = append(activeIncidents, entry)
				break
			}
		}
	}

	result := &StatusResult{
		Status:         StatusOK,
		ActiveIncidents: activeIncidents,
	}

	if len(activeIncidents) == 0 {
		result.Message = "No active incidents"
	} else if len(activeIncidents) == 1 {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("1 active incident: %s", activeIncidents[0].Title)
	} else {
		result.Status = StatusCritical
		result.Message = fmt.Sprintf("%d active incidents", len(activeIncidents))
	}

	return result
}
