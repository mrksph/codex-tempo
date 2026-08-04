package wakapi

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type Credentials struct {
	APIURL string
	APIKey string
}

type Interval struct {
	Project   string
	StartedAt time.Time
	EndedAt   time.Time
}

type Export struct {
	From       string
	To         string
	Heartbeats int
	Intervals  []Interval
	Duration   time.Duration
}

type Client struct {
	Credentials Credentials
	HTTP        *http.Client
}

func LoadCredentials(path string) (Credentials, error) {
	file, err := os.Open(path)
	if err != nil {
		return Credentials{}, err
	}
	defer file.Close()
	var credentials Credentials
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "api_url":
			credentials.APIURL = strings.TrimSpace(value)
		case "api_key":
			credentials.APIKey = strings.TrimSpace(value)
		}
	}
	if err = scanner.Err(); err != nil {
		return Credentials{}, err
	}
	if credentials.APIURL == "" || credentials.APIKey == "" {
		return Credentials{}, errors.New("wakatime config must contain api_url and api_key")
	}
	return credentials, nil
}

func (c Client) Export(ctx context.Context, from, to string, timeout time.Duration) (Export, error) {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	var allTime struct {
		Data struct {
			Range struct {
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
			} `json:"range"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/users/current/all_time_since_today", &allTime); err != nil {
		return Export{}, err
	}
	if from == "" {
		from = allTime.Data.Range.StartDate
	}
	if to == "" {
		to = allTime.Data.Range.EndDate
	}
	start, err := time.Parse(time.DateOnly, from)
	if err != nil {
		return Export{}, fmt.Errorf("invalid from date: %w", err)
	}
	end, err := time.Parse(time.DateOnly, to)
	if err != nil {
		return Export{}, fmt.Errorf("invalid to date: %w", err)
	}
	if end.Before(start) {
		return Export{}, errors.New("to date must not be before from date")
	}

	result := Export{From: from, To: to, Intervals: make([]Interval, 0)}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		var response struct {
			Data []struct {
				ID      string  `json:"id"`
				Time    float64 `json:"time"`
				Project string  `json:"project"`
			} `json:"data"`
		}
		path := "/users/current/heartbeats?date=" + url.QueryEscape(day.Format(time.DateOnly))
		if err = c.getJSON(ctx, path, &response); err != nil {
			return Export{}, fmt.Errorf("export %s: %w", day.Format(time.DateOnly), err)
		}
		sort.SliceStable(response.Data, func(i, j int) bool { return response.Data[i].Time < response.Data[j].Time })
		result.Heartbeats += len(response.Data)
		for i := 0; i+1 < len(response.Data); i++ {
			gap := response.Data[i+1].Time - response.Data[i].Time
			if gap <= 0 || gap >= timeout.Seconds() {
				continue
			}
			project := strings.TrimSpace(response.Data[i].Project)
			if project == "" {
				project = "unknown"
			}
			startedAt := unixTime(response.Data[i].Time)
			endedAt := unixTime(response.Data[i+1].Time)
			last := len(result.Intervals) - 1
			if last >= 0 && result.Intervals[last].Project == project && result.Intervals[last].EndedAt.Equal(startedAt) {
				result.Intervals[last].EndedAt = endedAt
			} else {
				result.Intervals = append(result.Intervals, Interval{Project: project, StartedAt: startedAt, EndedAt: endedAt})
			}
			result.Duration += endedAt.Sub(startedAt)
		}
	}
	return result, nil
}

func (c Client) getJSON(ctx context.Context, path string, target any) error {
	base := compatibleAPIURL(c.Credentials.APIURL)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
		if err != nil {
			return err
		}
		encodedKey := base64.StdEncoding.EncodeToString([]byte(c.Credentials.APIKey))
		request.Header.Set("Authorization", "Basic "+encodedKey)
		request.Header.Set("Accept", "application/json")
		response, err := httpClient.Do(request)
		if err == nil {
			if response.StatusCode/100 == 2 {
				err = json.NewDecoder(response.Body).Decode(target)
				response.Body.Close()
				return err
			}
			lastErr = fmt.Errorf("wakapi returned %s", response.Status)
			response.Body.Close()
			if response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests {
				return lastErr
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}
	return lastErr
}

func compatibleAPIURL(value string) string {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if strings.HasSuffix(value, "/compat/wakatime/v1") || strings.HasSuffix(value, "/api/v1") {
		return value
	}
	if strings.HasSuffix(value, "/api") {
		return value + "/compat/wakatime/v1"
	}
	return value + "/api/compat/wakatime/v1"
}

func unixTime(value float64) time.Time {
	seconds, fraction := math.Modf(value)
	return time.Unix(int64(seconds), int64(math.Round(fraction*float64(time.Second)))).UTC()
}
