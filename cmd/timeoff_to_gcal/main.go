package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"ooo-calendar-sync/utils"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
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

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("read stdin: %v", err)
	}
	var env envelope

	if err := json.Unmarshal(data, &env); err != nil {
		log.Fatalf("decode JSON: %v", err)
	}
	if len(env.Requests) == 0 {
		fmt.Println("No requests to process.")
		return
	}

	ctx := context.Background()
	b, err := os.ReadFile("service-account.json")
	if err != nil {
		log.Fatalf("read service-account.json: %v", err)
	}
	jwtCfg, err := google.JWTConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("JWT config: %v", err)
	}

	calendarIDs := []string{"primary"}

	for _, r := range env.Requests {
		// Load user's local times.
		loc, err := time.LoadLocation(r.UserTimeZone)
		if err != nil {
			log.Printf("skip %s: unknown tz %q: %v", r.ID, r.UserTimeZone, err)
			continue
		}

		startUTC, err := utils.ParseTimeAny(r.TimeOffPeriod.Period.Start)
		if err != nil {
			log.Printf("skip %s: bad period.start: %v", r.ID, err)
			continue
		}
		endUTC, err := utils.ParseTimeAny(r.TimeOffPeriod.Period.End)
		if err != nil {
			log.Printf("skip %s: bad period.end: %v", r.ID, err)
			continue
		}

		startLocal := startUTC.In(loc)
		endLocal := endUTC.In(loc)

		y1, m1, d1 := startLocal.Date()
		y2, m2, d2 := endLocal.Date()

		startDate := time.Date(y1, m1, d1, 0, 0, 0, 0, loc).Format("2006-01-02")
		// Clockify is inclusive; GCal all-day is [start, end) exclusive.
		// So cover the last OOO day by adding +1 local day to the end date.
		endDate := time.Date(y2, m2, d2, 0, 0, 0, 0, loc).AddDate(0, 0, 1).Format("2006-01-02")

		cfg := *jwtCfg
		cfg.Subject = r.UserEmail
		client := cfg.Client(ctx)

		srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Printf("user %s: calendar service error: %v", r.UserEmail, err)
			continue
		}

		summary := "[TEST] OOO"
		if r.PolicyName != "" {
			summary = fmt.Sprintf("[TEST] OOO — %s", r.PolicyName)
		}
		ev := &calendar.Event{
			Summary:     summary,
			Description: fmt.Sprintf("Clockify request: %s\nCreatedAt: %s", r.ID, r.CreatedAt),
			Start:       &calendar.EventDateTime{Date: startDate},
			End:         &calendar.EventDateTime{Date: endDate}, // exclusive
			// Attaching the Clockify request ID as a private extended property.
			// TODO: Before inserting, check for an existing event with this key and insert event/skip:
			ExtendedProperties: &calendar.EventExtendedProperties{
				Private: map[string]string{
					"clockifyRequestId": r.ID,
				},
			},
		}

		for _, calID := range calendarIDs {
			_, err := srv.Events.Insert(calID, ev).Do()
			if err != nil {
				log.Printf("insert %s (user=%s cal=%s) failed: %v", r.ID, r.UserEmail, calID, err)
				continue
			}
			fmt.Printf("Inserted OOO for req=%s user=%s cal=%s (%s → %s)\n",
				r.ID, r.UserEmail, calID, startDate, endDate)
		}
	}
}
