// Package catalogclient lets activity-service enrich a bare
// collection.record_added event (which only carries IDs) with the record's
// genre, by calling catalog-service synchronously — the same "call the
// service that actually owns this data" pattern collection-service uses,
// just triggered from an event consumer instead of an HTTP handler.
package catalogclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RecordSummary struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Genre  string `json:"genre"`
	Year   int    `json:"year"`
}

type RecordFetcher interface {
	GetRecord(ctx context.Context, recordID string) (RecordSummary, error)
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *Client) GetRecord(ctx context.Context, recordID string) (RecordSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/records/"+recordID, nil)
	if err != nil {
		return RecordSummary{}, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return RecordSummary{}, fmt.Errorf("calling catalog-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RecordSummary{}, fmt.Errorf("catalog-service returned %d for record %s", resp.StatusCode, recordID)
	}

	var summary RecordSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return RecordSummary{}, fmt.Errorf("decode catalog response: %w", err)
	}
	return summary, nil
}
