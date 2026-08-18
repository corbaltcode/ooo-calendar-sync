package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/corbaltcode/ooo-calendar-sync/core"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

type Event struct {
	PeriodStart   string `json:"start"`
	PeriodEnd     string `json:"end"`
	ActivityStart string `json:"activityStart"`
	ActivityEnd   string `json:"activityEnd"`
	FilterBy      string `json:"by"`
	PageSize      int    `json:"pageSize"`
}

func (e *Event) Run(ctx context.Context) {
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

	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		core.Die("missing env DYNAMODB_TABLE_NAME")
	}

	if e.PageSize <= 0 {
		core.Die("invalid pageSize: must be > 0")
	}

	if e.FilterBy == "" {
		core.Die("missing required parameter: by")
	}

	validFilterBys := map[string]bool{"period": true, "activity": true}
	if !validFilterBys[e.FilterBy] {
		core.Die("invalid -by: must be 'period' or 'activity'")
	}

	var startPtr, endPtr *string
	if e.PeriodStart != "" {
		ts, err := core.ParseAndFormatClockifyTime(e.PeriodStart)
		if err != nil {
			core.Die("invalid start time: %v", err)
		}
		startPtr = &ts
	}
	if e.PeriodEnd != "" {
		ts, err := core.ParseAndFormatClockifyTime(e.PeriodEnd)
		if err != nil {
			core.Die("invalid end time: %v", err)
		}
		endPtr = &ts
	}

	payload := core.ClockifyRequestPayload{
		Start:    startPtr,
		End:      endPtr,
		Page:     1,
		PageSize: e.PageSize,
		Statuses: []string{
			core.ClockifyStatusApproved,
			core.ClockifyStatusRejected,
		},
	}

	// Development safety: force a single user via env var, if set.
	if forcedSingleUser := os.Getenv("CLOCKIFY_FORCE_USER_ID"); forcedSingleUser != "" {
		fmt.Printf("CLOCKIFY_FORCE_USER_ID active: only syncing user %s\n", forcedSingleUser)
		payload.Users = []string{forcedSingleUser}
	}

	if e.FilterBy == "activity" && (payload.Start == nil || payload.End == nil) {
		core.Die("when -by=activity is used, both -start and -end must be provided")
	}

	client := core.NewClockifyClient(apiKey)

	respBytes, err := core.FetchClockifyRequests(client, workspaceID, payload)
	if err != nil {
		core.Die("fetch clockify: %v", err)
	}

	var activityStartT, activityEndT time.Time
	var activityStartOK, activityEndOK bool

	if e.ActivityStart != "" {
		t, err := core.ParseFlexibleRFC3339(e.ActivityStart)
		if err != nil {
			core.Die("invalid activityStart: %v", err)
		}
		activityStartT, activityStartOK = t.UTC(), true
	}

	if e.ActivityEnd != "" {
		t, err := core.ParseFlexibleRFC3339(e.ActivityEnd)
		if err != nil {
			core.Die("invalid activityEnd: %v", err)
		}
		activityEndT, activityEndOK = t.UTC(), true
	}

	// Print results and early return if not filtering by activity.
	if e.FilterBy != "activity" || (!activityStartOK && !activityEndOK) {
		if pretty, err := core.PrettyJSON(respBytes); err == nil {
			fmt.Println(pretty)
		} else {
			fmt.Println(string(respBytes))
		}
		return
	}

	// TODO: Revisit the naming of the time window variables now that filtering
	// includes both request creation and status changes.
	env, err := core.FilterByActivity(respBytes, activityStartT, activityEndT)
	if err != nil {
		core.Die("filter: %v", err)
	}

	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		core.Die("load AWS config: %v", err)
	}

	store := core.NewDynamoStore(
		dynamodb.NewFromConfig(awsCfg),
		tableName,
	)

	// TODO: Move request filtering into the core package once the persistence layer is fully implemented.
	var requestsToProcess []core.RequestToProcess

	for _, req := range env.Requests {
		existing, err := store.GetSyncedRequest(ctx, req.ID)
		if err != nil {
			core.Die("get synced request %s: %v", req.ID, err)
		}

		currentStatus := req.Status.StatusType

		if existing == nil {
			log.Printf(
				"Queueing new Clockify request %s with status %s",
				req.ID,
				currentStatus,
			)

			requestsToProcess = append(requestsToProcess, core.RequestToProcess{
				Request: req,
			})
			continue
		}

		if existing.Status != currentStatus {
			log.Printf(
				"Queueing Clockify request %s because status changed from %s to %s",
				req.ID,
				existing.Status,
				currentStatus,
			)

			requestsToProcess = append(requestsToProcess, core.RequestToProcess{
				Request:        req,
				ExistingRecord: existing,
			})
			continue
		}

		log.Printf(
			"Skipping Clockify request %s because status %s has already been processed",
			req.ID,
			currentStatus,
		)
	}

	if len(requestsToProcess) == 0 {
		fmt.Println("No requests queued for processing.")
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
	var syncErrs []error

	for _, req := range requestsToProcess {
		calendarEvents, err := core.SyncOOORequest(
			ctx,
			*jwtCfg,
			req,
			calendarIDs,
		)
		if err != nil {
			log.Printf(
				"Failed to sync Clockify request %s: %v",
				req.Request.ID,
				err,
			)

			syncErrs = append(
				syncErrs,
				fmt.Errorf("sync request %s: %w", req.Request.ID, err),
			)

			continue
		}

		log.Printf(
			"Successfully synced Clockify request %s to Google Calendar",
			req.Request.ID,
		)

		dynamoItem, err := req.Request.ToDynamoItem()

		if err != nil {
			log.Printf(
				"Failed to convert Clockify request %s to a DynamoDB item: %v",
				req.Request.ID,
				err,
			)

			syncErrs = append(
				syncErrs,
				fmt.Errorf("convert request %s to DynamoDB item: %w", req.Request.ID, err),
			)
			continue
		}

		dynamoItem.SyncState = "synced"
		dynamoItem.GoogleCalendarEvents = calendarEvents

		if err := store.PutSyncedRequest(ctx, dynamoItem); err != nil {
			log.Printf(
				"Failed to store Clockify request %s in DynamoDB: %v",
				req.Request.ID,
				err,
			)

			syncErrs = append(
				syncErrs,
				fmt.Errorf("store request %s in DynamoDB: %w", req.Request.ID, err),
			)
			continue
		}

		log.Printf(
			"Successfully stored Clockify request %s in DynamoDB",
			req.Request.ID,
		)
	}

	if err := errors.Join(syncErrs...); err != nil {
		core.Die("sync completed with errors: %v", err)
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

	ev.Run(ctx)
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
		periodStartStr   = flag.String("start", "", "Period start (RFC3339)")
		periodEndStr     = flag.String("end", "", "Period end (RFC3339)")
		filterBy         = flag.String("by", "activity", "Filter mode: period|activity")
		activityStartStr = flag.String("activityStart", "", "Created or updated >= (RFC3339)")
		activityEndStr   = flag.String("activityEnd", "", "Created or updated < (RFC3339)")
		pageSize         = flag.Int("pageSize", 50, "Page size (1–200)")
	)

	flag.Parse()
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: no .env file found, relying on environment vars")
	}

	ev := Event{
		PeriodStart:   *periodStartStr,
		PeriodEnd:     *periodEndStr,
		ActivityStart: *activityStartStr,
		ActivityEnd:   *activityEndStr,
		FilterBy:      *filterBy,
		PageSize:      *pageSize,
	}

	ev.Run(context.Background())
}
