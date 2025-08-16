package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
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
		startStr        = flag.String("start", "", "Start (RFC3339-ish, e.g. 2025-08-01T00:00:00Z) — optional")
		endStr          = flag.String("end", "", "End (RFC3339-ish, e.g. 2025-08-10T23:59:59Z) — optional")
		statusesStr     = flag.String("statuses", "APPROVED", "Comma-separated statuses: PENDING,APPROVED,REJECTED,ALL")
		page            = flag.Int("page", 1, "Page number (default 1)")
		pageSize        = flag.Int("pageSize", 50, "Page size (1..200)")
		by              = flag.String("by", "period", `Date filter mode: "period" (default) or "created"`)
		createdStartStr = flag.String("createdStart", "", "Filter by createdAt >= this instant (RFC3339 or date-only)")
		createdEndStr   = flag.String("createdEnd", "", "Filter by createdAt <  this instant (RFC3339 or date-only)")
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

	// Parsing the createdAt filters.
	var createdStart, createdEnd time.Time
	var createdStartOK, createdEndOK bool
	if *createdStartStr != "" {
		t, err := parseFlexibleRFC3339(*createdStartStr)
		if err != nil {
			log.Fatalf("invalid -createdStart: %v", err)
		}
		createdStart, createdStartOK = t.UTC(), true
	}
	if *createdEndStr != "" {
		t, err := parseFlexibleRFC3339(*createdEndStr)
		if err != nil {
			log.Fatalf("invalid -createdEnd: %v", err)
		}
		createdEnd, createdEndOK = t.UTC(), true
	}

	// Build request payload.
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

	if *by == "created" && (payload.Start == nil || payload.End == nil) {
		die("when -by=created is used, both -start and -end must be provided")
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

	// Print results and early return if not filtering by `createdAt`.
	if *by != "created" || (!createdStartOK && !createdEndOK) {
		if pretty, err := prettyJSON(respBytes); err == nil {
			fmt.Println(pretty)
			return
		}
		fmt.Println(string(respBytes))
		return
	}

	// Filter (in memory) by createdAt property.
	// Parse only enough to filter, but keep original objects intact.
	type createdOnly struct {
		CreatedAt string `json:"createdAt"`
	}
	type apiResp struct {
		Count    int               `json:"count"`
		Requests []json.RawMessage `json:"requests"`
	}
	var ar apiResp
	if err := json.Unmarshal(respBytes, &ar); err != nil {
		die("decode: %v", err)
	}

	filtered := make([]json.RawMessage, 0, len(ar.Requests))
	for _, raw := range ar.Requests {
		var c createdOnly
		if err := json.Unmarshal(raw, &c); err != nil {
			// If a record can't be parsed, skip
			continue
		}
		// Parse createdAt
		ct, err := parseFlexibleRFC3339(c.CreatedAt)
		if err != nil {
			continue
		}
		ct = ct.UTC()

		if createdStartOK && ct.Before(createdStart) {
			continue
		}
		// Treat createdEnd as exclusive; use `!ct.After(createdEnd)`` for inclusive.
		if createdEndOK && !ct.Before(createdEnd) {
			continue
		}
		filtered = append(filtered, raw)
	}

	out := struct {
		Count    int               `json:"count"`
		Requests []json.RawMessage `json:"requests"`
	}{
		Count:    len(filtered),
		Requests: filtered,
	}
	outBytes, _ := json.Marshal(out)

	if pretty, err := prettyJSON(outBytes); err == nil {
		fmt.Println(pretty)
		return
	}
	fmt.Println(string(outBytes))
}

// Utils
func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

var acceptedLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02",
}

// Tries to parse multiple date/time layouts and returns a UTC time.
func parseTimeAny(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range acceptedLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

func parseFlexibleRFC3339(s string) (time.Time, error) {
	return parseTimeAny(s)
}

// Clockify expects YYYY-MM-DDTHH:MM:SS.ssssssZ (microseconds).
// https://docs.clockify.me/#tag/Time-Off/operation/getTimeOffRequest
func formatClockify(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func parseAndFormatClockifyTime(s string) (string, error) {
	t, err := parseTimeAny(s)
	if err != nil {
		return "", fmt.Errorf("cannot parse time %q: %w", s, err)
	}
	return formatClockify(t), nil
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
