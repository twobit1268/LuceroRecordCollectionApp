// Package catalogclient is the synchronous side of this system's
// distributed-ness: before adding a record to a customer's collection,
// collection-service calls catalog-service over plain REST to confirm the
// record actually exists, rather than trusting a client-supplied recordId.
package catalogclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RecordValidator is the interface handlers depend on, so tests can inject
// a fake instead of making real HTTP calls to catalog-service.
type RecordValidator interface {
	RecordExists(ctx context.Context, recordID string) (bool, error)
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) RecordExists(ctx context.Context, recordID string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/records/"+recordID, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("calling catalog-service: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		var body map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return false, fmt.Errorf("catalog-service returned %d: %v", resp.StatusCode, body)
	}
}
