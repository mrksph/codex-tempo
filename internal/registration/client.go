package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ServerURL, SetupKey string
	HTTP                *http.Client
}

type Result struct {
	MachineID string `json:"machine_id"`
	Token     string `json:"token"`
}

func (c Client) Register(ctx context.Context, machineID, name string) (Result, error) {
	var result Result
	body, err := json.Marshal(map[string]string{"machine_id": machineID, "name": name})
	if err != nil {
		return result, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.ServerURL, "/")+"/api/v1/ingest/register", bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	request.Header.Set("Authorization", "Bearer "+c.SetupKey)
	request.Header.Set("Content-Type", "application/json")
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		var apiError struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&apiError)
		if apiError.Error == "" {
			apiError.Error = response.Status
		}
		return result, fmt.Errorf("register machine: %s", apiError.Error)
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, err
	}
	if result.MachineID != machineID || result.Token == "" {
		return result, fmt.Errorf("register machine: invalid server response")
	}
	return result, nil
}
