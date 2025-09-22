package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/corbaltcode/ooo-calendar-sync/core"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func main() {
	var (
		periodStartStr  = flag.String("start", "", "Start of time-off period (RFC3339-ish, e.g. 2025-08-01T00:00:00Z)")
		periodEndStr    = flag.String("end", "", "End of time-off period (RFC3339-ish, e.g. 2025-08-10T23:59:59Z)")
		statusesStr     = flag.String("statuses", "APPROVED", "Comma-separated statuses: PENDING,APPROVED,REJECTED,ALL")
		startPage       = flag.Int("page", 1, "Page number (default 1)")
		pageSize        = flag.Int("pageSize", 50, "Page size (1..200)")
		filterBy        = flag.String("by", "period", `Date filter mode: "period" (default) or "created"`)
		createdStartStr = flag.String("createdStart", "", "Filter time-off requests creation >= this instant (RFC3339 or date-only)")
		createdEndStr   = flag.String("createdEnd", "", "Filter time-off requests creation <  this instant (RFC3339 or date-only)")
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
			core.Die("invalid -createdStart: %v", err)
		}
		createdStart, createdStartOK = t.UTC(), true
	}
	if *createdEndStr != "" {
		t, err := core.ParseFlexibleRFC3339(*createdEndStr)
		if err != nil {
			core.Die("invalid -createdEnd: %v", err)
		}
		createdEnd, createdEndOK = t.UTC(), true
	}

	if *startPage <= 0 {
		core.Die("invalid -page: must be > 0")
	}

	if *pageSize <= 0 {
		core.Die("invalid -pageSize: must be > 0")
	}

	validFilterBys := map[string]bool{"period": true, "created": true}
	if !validFilterBys[*filterBy] {
		core.Die("invalid -by: must be 'period' or 'created'")
	}

	validStatuses := map[string]bool{
		"PENDING":  true,
		"APPROVED": true,
		"REJECTED": true,
		"ALL":      true,
	}

	var statuses []string
	for _, s := range strings.Split(*statusesStr, ",") {
		s = strings.ToUpper(strings.TrimSpace(s))

		if !validStatuses[s] {
			core.Die("invalid -statuses value: %q (must be PENDING, APPROVED, REJECTED, ALL)", s)
		}
		statuses = append(statuses, s)
	}

	// Build request payload.
	var startPtr, endPtr *string
	if *periodStartStr != "" {
		ts, err := core.ParseAndFormatClockifyTime(*periodStartStr)
		if err != nil {
			core.Die("invalid -start time: %v", err)
		}
		startPtr = &ts
	}
	if *periodEndStr != "" {
		ts, err := core.ParseAndFormatClockifyTime(*periodEndStr)
		if err != nil {
			core.Die("invalid -end time: %v", err)
		}
		endPtr = &ts
	}

	payload := core.ClockifyRequestPayload{
		Start:    startPtr,
		End:      endPtr,
		Page:     *startPage,
		PageSize: *pageSize,
		Statuses: statuses,
	}

	if *filterBy == "created" && (payload.Start == nil || payload.End == nil) {
		core.Die("when -by=created is used, both -start and -end must be provided")
	}

	client := core.NewClockifyClient(apiKey)

	respBytes, err := core.FetchClockifyRequests(client, workspaceID, payload)
	if err != nil {
		core.Die("fetch clockify: %v", err)
	}

	// Print results and early return if not filtering by `createdAt`.
	if *filterBy != "created" || (!createdStartOK && !createdEndOK) {
		if pretty, err := core.PrettyJSON(respBytes); err == nil {
			fmt.Println(pretty)
			return
		}
		fmt.Println(string(respBytes))
		return
	}

	env, err := core.FilterByCreatedAt(respBytes, createdStart, createdEnd)
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
