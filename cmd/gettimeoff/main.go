package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type requestPayload struct {
	Start    *string  `json:"start,omitempty"`
	End      *string  `json:"end,omitempty"`
	Page     int      `json:"page,omitempty"`
	PageSize int      `json:"pageSize,omitempty"`
	Statuses []string `json:"statuses,omitempty"`
}

func main() {
	var (
		startStr    = flag.String("start", "", "Start (RFC3339-ish, e.g. 2025-08-01T00:00:00Z) — optional")
		endStr      = flag.String("end", "", "End (RFC3339-ish, e.g. 2025-08-10T23:59:59Z) — optional")
		statusesStr = flag.String("statuses", "APPROVED", "Comma-separated statuses: PENDING,APPROVED,REJECTED,ALL")
		page        = flag.Int("page", 1, "Page number (default 1)")
		pageSize    = flag.Int("pageSize", 50, "Page size (1..200)")
	)
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: no .env file found, relying on shell environment")
	}

	apiKey := os.Getenv("CLOCKIFY_API_KEY")
	if apiKey == "" {
		die("missing env CLOCKIFY_API_KEY")
	}

	workspaceID := os.Getenv("WORKSPACE_ID")
	if workspaceID == "" {
		die("missing env WORKSPACE_ID")
	}

	var startPtr, endPtr *string
	if *startStr != "" {
		ts, err := parseAndFormatClockifyTime(*startStr)
		if err != nil {
			die("invalid -start time: %v", err)
		}
		startPtr = &ts
	}
	if *endStr != "" {
		ts, err := parseAndFormatClockifyTime(*endStr)
		if err != nil {
			die("invalid -end time: %v", err)
		}
		endPtr = &ts
	}

	var statuses []string
	for _, s := range strings.Split(*statusesStr, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			statuses = append(statuses, strings.ToUpper(s))
		}
	}

	payload := requestPayload{
		Start:    startPtr,
		End:      endPtr,
		Page:     *page,
		PageSize: *pageSize,
		Statuses: statuses,
	}

	url := fmt.Sprintf("https://api.clockify.me/api/v1/workspaces/%s/time-off/requests", workspaceID)
	body, err := json.Marshal(payload)
	if err != nil {
		die("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		die("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		die("http request: %v", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		die("read body: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		die("non-2xx status: %s\n%s", resp.Status, string(respBytes))
	}

	pretty, err := prettyJSON(respBytes)
	if err != nil {
		// If it isn't JSON, just print raw data
		fmt.Println(string(respBytes))
		return
	}

	fmt.Println(pretty)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

// Clockify expects YYYY-MM-DDTHH:MM:SS.ssssssZ (microseconds).
// https://docs.clockify.me/#tag/Time-Off/operation/getTimeOffRequest
func parseAndFormatClockifyTime(s string) (string, error) {
	// Try common layouts first
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02", // date only
	}
	var t time.Time
	var err error
	for _, l := range layouts {
		t, err = time.Parse(l, s)
		if err == nil {
			utc := t.UTC()
			return utc.Format("2006-01-02T15:04:05.000000Z"), nil
		}
	}
	return "", fmt.Errorf("cannot parse time %q", s)
}

func prettyJSON(b []byte) (string, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
