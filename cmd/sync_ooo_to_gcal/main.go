package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"ooo-calendar-sync/core"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type envelope struct {
	Requests []request `json:"requests"`
}
type request struct {
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
		core.Die("missing env CLOCKIFY_API_KEY")
	}

	workspaceID := os.Getenv("WORKSPACE_ID")
	if workspaceID == "" {
		core.Die("missing env WORKSPACE_ID")
	}

	// Parsing the createdAt filters.
	var createdStart, createdEnd time.Time

	var createdStartOK, createdEndOK bool
	if *createdStartStr != "" {
		t, err := core.ParseFlexibleRFC3339(*createdStartStr)
		if err != nil {
			log.Fatalf("invalid -createdStart: %v", err)
		}
		createdStart, createdStartOK = t.UTC(), true
	}
	if *createdEndStr != "" {
		t, err := core.ParseFlexibleRFC3339(*createdEndStr)
		if err != nil {
			log.Fatalf("invalid -createdEnd: %v", err)
		}
		createdEnd, createdEndOK = t.UTC(), true
	}

	// Build request payload.
	var startPtr, endPtr *string
	if *startStr != "" {
		ts, err := core.ParseAndFormatClockifyTime(*startStr)
		if err != nil {
			core.Die("invalid -start time: %v", err)
		}
		startPtr = &ts
	}
	if *endStr != "" {
		ts, err := core.ParseAndFormatClockifyTime(*endStr)
		if err != nil {
			core.Die("invalid -end time: %v", err)
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

	payload := core.RequestPayload{
		Start:    startPtr,
		End:      endPtr,
		Page:     *page,
		PageSize: *pageSize,
		Statuses: statuses,
	}

	if *by == "created" && (payload.Start == nil || payload.End == nil) {
		core.Die("when -by=created is used, both -start and -end must be provided")
	}

	respBytes, err := core.FetchClockifyRequests(apiKey, workspaceID, payload)
	if err != nil {
		core.Die("fetch clockify: %v", err)
	}

	// Print results and early return if not filtering by `createdAt`.
	if *by != "created" || (!createdStartOK && !createdEndOK) {
		if pretty, err := core.PrettyJSON(respBytes); err == nil {
			fmt.Println(pretty)
			return
		}
		fmt.Println(string(respBytes))
		return
	}

	env, err := core.FilterByCreatedAt(respBytes, &createdStart, &createdEnd)
	if err != nil {
		core.Die("filter: %v", err)
	}

	if len(env.Requests) == 0 {
		fmt.Println("No requests to process.")
		return
	}

	ctx := context.Background()
	b, err := os.ReadFile("service-account.json")
	if err != nil {
		core.Die("read service-account.json: %v", err)
	}

	jwtCfg, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		core.Die("JWT config: %v", err)
	}

	calendarIDs := []string{"primary"}
	if err := core.InsertOOOEvents(ctx, jwtCfg, env.Requests, calendarIDs); err != nil {
		core.Die("insert events: %v", err)
	}
}
