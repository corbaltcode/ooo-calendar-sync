package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/corbaltcode/ooo-calendar-sync/core"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type Event struct {
	Start        string   `json:"start"`
	End          string   `json:"end"`
	CreatedStart string   `json:"createdStart"`
	CreatedEnd   string   `json:"createdEnd"`
	Statuses     []string `json:"statuses"`
	By           string   `json:"by"`
	PageSize     int      `json:"pageSize"`
}

func run(ctx context.Context, periodStart, periodEnd, createdStart, createdEnd string, statuses []string, filterBy string, pageSize int) {
	apiKey := os.Getenv("CLOCKIFY_API_KEY")
	if apiKey == "" {
		core.Die("missing env CLOCKIFY_API_KEY")
	}
	workspaceID := os.Getenv("WORKSPACE_ID")
	if workspaceID == "" {
		core.Die("missing env WORKSPACE_ID")
	}
	credB64 := os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON_B64")
	if credB64 == "" {
		core.Die("missing env GOOGLE_SERVICE_ACCOUNT_JSON_B64")
	}

	if pageSize <= 0 {
		core.Die("invalid pageSize: must be > 0")
	}

	if filterBy == "" {
		core.Die("missing required parameter: by")
	}

	validFilterBys := map[string]bool{"period": true, "created": true}
	if !validFilterBys[filterBy] {
		core.Die("invalid -by: must be 'period' or 'created'")
	}

	var startPtr, endPtr *string
	if periodStart != "" {
		ts, err := core.ParseAndFormatClockifyTime(periodStart)
		if err != nil {
			core.Die("invalid start time: %v", err)
		}
		startPtr = &ts
	}
	if periodEnd != "" {
		ts, err := core.ParseAndFormatClockifyTime(periodEnd)
		if err != nil {
			core.Die("invalid end time: %v", err)
		}
		endPtr = &ts
	}

	validStatuses := map[string]bool{
		"PENDING":  true,
		"APPROVED": true,
		"REJECTED": true,
		"ALL":      true,
	}

	if len(statuses) == 0 {
		core.Die("missing or empty statuses list")
	}

	for _, s := range statuses {
		s = strings.ToUpper(strings.TrimSpace(s))
		if !validStatuses[s] {
			core.Die("invalid statuses value: %q (must be one of PENDING, APPROVED, REJECTED, ALL)", s)
		}
	}

	payload := core.ClockifyRequestPayload{
		Start:    startPtr,
		End:      endPtr,
		Page:     1,
		PageSize: pageSize,
		Statuses: statuses,
	}

	if filterBy == "created" && (payload.Start == nil || payload.End == nil) {
		core.Die("when -by=created is used, both -start and -end must be provided")
	}

	client := core.NewClockifyClient(apiKey)

	respBytes, err := core.FetchClockifyRequests(client, workspaceID, payload)
	if err != nil {
		core.Die("fetch clockify: %v", err)
	}

	var createdStartT, createdEndT time.Time
	var createdStartOK, createdEndOK bool

	if createdStart != "" {
		t, err := core.ParseFlexibleRFC3339(createdStart)
		if err != nil {
			core.Die("invalid createdStart: %v", err)
		}
		createdStartT, createdStartOK = t.UTC(), true
	}

	if createdEnd != "" {
		t, err := core.ParseFlexibleRFC3339(createdEnd)
		if err != nil {
			core.Die("invalid createdEnd: %v", err)
		}
		createdEndT, createdEndOK = t.UTC(), true
	}

	// Print results and early return if not filtering by createdAt.
	if filterBy != "created" || (!createdStartOK && !createdEndOK) {
		if pretty, err := core.PrettyJSON(respBytes); err == nil {
			fmt.Println(pretty)
		} else {
			fmt.Println(string(respBytes))
		}
		return
	}

	env, err := core.FilterByCreatedAt(respBytes, createdStartT, createdEndT)
	if err != nil {
		core.Die("filter: %v", err)
	}
	if len(env.Requests) == 0 {
		fmt.Println("No requests to process.")
		return
	}

	b, err := base64.StdEncoding.DecodeString(credB64)
	if err != nil {
		core.Die("invalid base64 GOOGLE_SERVICE_ACCOUNT_JSON_B64: %v", err)
	}

	jwtCfg, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		core.Die("JWT config: %v", err)
	}

	calendarIDs := []string{"primary"}
	if err := core.InsertOOOEvents(ctx, jwtCfg, env.Requests, calendarIDs); err != nil {
		core.Die("insert events: %v", err)
	}

	fmt.Println("Sync complete!")
}

func handler(ctx context.Context, e json.RawMessage) error {
	var ev Event
	if len(e) > 0 {
		if err := json.Unmarshal(e, &ev); err != nil {
			core.Die("invalid JSON event: %v", err)
		}
	}

	run(ctx, ev.Start, ev.End, ev.CreatedStart, ev.CreatedEnd, ev.Statuses, ev.By, ev.PageSize)
	return nil
}

func main() {
	// If we're on Lambda runtime
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(handler)
		return
	}

	// CLI mode
	var (
		periodStartStr  = flag.String("start", "", "Period start (RFC3339)")
		periodEndStr    = flag.String("end", "", "Period end (RFC3339)")
		statusesStr     = flag.String("statuses", "APPROVED", "Comma-separated statuses")
		filterBy        = flag.String("by", "created", "Filter mode: period|created")
		createdStartStr = flag.String("createdStart", "", "Created >= (RFC3339)")
		createdEndStr   = flag.String("createdEnd", "", "Created <  (RFC3339)")
		pageSize        = flag.Int("pageSize", 50, "Page size (1–200)")
	)
	flag.Parse()
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: no .env file found, relying on environment vars")
	}

	var statuses []string
	for _, s := range strings.Split(*statusesStr, ",") {
		statuses = append(statuses, strings.ToUpper(strings.TrimSpace(s)))
	}

	run(context.Background(), *periodStartStr, *periodEndStr, *createdStartStr, *createdEndStr, statuses, *filterBy, *pageSize)
}
