package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/mrksph/codex-tempo/internal/domain"
	"github.com/mrksph/codex-tempo/internal/localdb"
)

type Client struct {
	Store                       *localdb.Store
	ServerURL, Token, MachineID string
	HTTP                        *http.Client
}
type ingestRequest struct {
	MachineID string         `json:"machine_id"`
	Events    []domain.Event `json:"events"`
}
type ingestResponse struct {
	Accepted   int                           `json:"accepted"`
	Duplicates int                           `json:"duplicates"`
	Rejected   []struct{ ID, Reason string } `json:"rejected"`
}

func (c *Client) SyncOnce(ctx context.Context) error {
	if c.ServerURL == "" {
		return nil
	}
	events, err := c.Store.Pending(ctx, 250)
	if err != nil || len(events) == 0 {
		return err
	}
	body, _ := json.Marshal(ingestRequest{MachineID: c.MachineID, Events: events})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.ServerURL, "/")+"/api/v1/ingest/events", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("ingest returned %s", resp.Status)
	}
	var result ingestResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	rejected := make(map[string]bool)
	for _, item := range result.Rejected {
		rejected[item.ID] = true
	}
	ids := make([]string, 0, len(events))
	for _, event := range events {
		if !rejected[event.ID] {
			ids = append(ids, event.ID)
		}
	}
	return c.Store.MarkAcknowledged(ctx, ids)
}

func (c *Client) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	delay := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			if err := c.SyncOnce(ctx); err != nil {
				if delay < interval {
					delay = interval
				}
				delay = min(delay*2, 10*time.Minute) + time.Duration(rand.IntN(1000))*time.Millisecond
			} else {
				pending, err := c.Store.Pending(ctx, 1)
				if err != nil || len(pending) == 0 {
					delay = interval
				} else {
					delay = 0
				}
			}
		}
	}
}
