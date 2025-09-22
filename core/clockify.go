package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ClockifyEnvelope struct {
	Requests []ClockifyRequest `json:"requests"`
}

type ClockifyRequest struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	PolicyName string `json:"policyName"`

	UserEmail    string `json:"userEmail"`
	UserTimeZone string `json:"userTimeZone"`

	TimeOffPeriod struct {
		Period struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"period"`
	} `json:"timeOffPeriod"`
}

type ClockifyRequestPayload struct {
	Start    *string  `json:"start,omitempty"`
	End      *string  `json:"end,omitempty"`
	Page     int      `json:"page,omitempty"`
	PageSize int      `json:"pageSize,omitempty"`
	Statuses []string `json:"statuses,omitempty"`
}

const defaultClockifyBaseURL = "https://api.clockify.me/api/v1"

type ClockifyClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClockifyClient(apiKey string, opts ...func(*ClockifyClient)) *ClockifyClient {
	client := &ClockifyClient{
		baseURL: defaultClockifyBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// For testing
func WithClockifyBaseURL(baseURL string) func(*ClockifyClient) {
	return func(c *ClockifyClient) {
		c.baseURL = baseURL
	}
}

func WithHTTPClient(h *http.Client) func(*ClockifyClient) {
	return func(c *ClockifyClient) {
		c.http = h
	}
}

func FetchClockifyRequests(c *ClockifyClient, workspaceID string, payload ClockifyRequestPayload) ([]byte, error) {
	url := fmt.Sprintf("%s/workspaces/%s/time-off/requests", c.baseURL, workspaceID)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("non-2xx status: %s\n%s", resp.Status, string(respBytes))
	}

	return respBytes, nil
}
